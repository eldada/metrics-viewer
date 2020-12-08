package provider

import (
	"encoding/base64"
	"fmt"
	"github.com/jfrog/jfrog-cli-core/artifactory/utils"
	"github.com/jfrog/jfrog-cli-core/utils/config"
	"github.com/jfrog/jfrog-client-go/artifactory/httpclient"
	"github.com/jfrog/jfrog-client-go/utils/io/httputils"
	"io/ioutil"
	"net/http"
	"strings"
)

type UrlMetricsFetcher interface {
	Get() ([]byte, error)
}

func NewArtifactoryMetricsFetcher(rtDetails *config.ArtifactoryDetails) (*artifactoryMetricsFetcher, error) {
	sm, err := utils.CreateServiceManager(rtDetails, false)
	if err != nil {
		return nil, err
	}
	authConfig, err := rtDetails.CreateArtAuthConfig()
	if err != nil {
		return nil, err
	}
	clientDetails := authConfig.CreateHttpClientDetails()
	return &artifactoryMetricsFetcher{
		url:           fmt.Sprintf("%s/api/v1/metrics", strings.TrimSuffix(rtDetails.Url, "/")),
		client:        sm.Client(),
		clientDetails: &clientDetails,
	}, nil
}

type artifactoryMetricsFetcher struct {
	url           string
	client        *httpclient.ArtifactoryHttpClient
	clientDetails *httputils.HttpClientDetails
}

func (f *artifactoryMetricsFetcher) Get() ([]byte, error) {
	res, body, _, err := f.client.SendGet(f.url, true, f.clientDetails)
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status: %s", res.Status)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("response body is empty")
	}
	return body, nil
}

func (f artifactoryMetricsFetcher) String() string {
	return fmt.Sprintf("url: %s, user: %s", f.url, f.clientDetails.User)
}

func NewUrlMetricsFetcher(url string, authenticator Authenticator) *urlMetricsFetcher {
	return &urlMetricsFetcher{
		url:           url,
		authenticator: authenticator,
	}
}

type urlMetricsFetcher struct {
	url           string
	authenticator Authenticator
}

func (f urlMetricsFetcher) String() string {
	if f.authenticator == nil {
		return fmt.Sprintf("url: %s", f.url)
	}
	return fmt.Sprintf("url: %s, auth-by-%s", f.url, f.authenticator)
}

func (f urlMetricsFetcher) Get() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, f.url, nil)
	if err != nil {
		return nil, err
	}
	if f.authenticator != nil {
		f.authenticator.Authorize(req)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status: %s", res.Status)
	}
	return ioutil.ReadAll(res.Body)
}

type Authenticator interface {
	Authorize(req *http.Request)
}

type UserPassAuthenticator struct {
	Username string
	Password string
}

func (a UserPassAuthenticator) Authorize(req *http.Request) {
	credsEncoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", a.Username, a.Password)))
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", credsEncoded))
}

func (a UserPassAuthenticator) String() string {
	return fmt.Sprintf("user: %s", a.Username)
}

type AccessTokenAuthenticator struct {
	Token string
}

func (a AccessTokenAuthenticator) Authorize(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}

func (a AccessTokenAuthenticator) String() string {
	return fmt.Sprintf("token: *****")
}
