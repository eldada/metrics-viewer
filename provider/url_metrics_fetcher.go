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
		return nil, fmt.Errorf("unexpected response status code: %d", res.StatusCode)
	}
	if len(body) == 0 {
		return nil, fmt.Errorf("response body is empty")
	}
	return body, nil
}

func (f artifactoryMetricsFetcher) String() string {
	return fmt.Sprintf("url: %s, user: %s", f.url, f.clientDetails.User)
}

func NewUrlMetricsFetcher(url, username, password string) *urlMetricsFetcher {
	return &urlMetricsFetcher{
		url:      url,
		username: username,
		password: password,
	}
}

type urlMetricsFetcher struct {
	url      string
	username string
	password string
}

func (f urlMetricsFetcher) String() string {
	return fmt.Sprintf("url: %s, user: %s", f.url, f.username)
}

func (f urlMetricsFetcher) Get() ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, f.url, nil)
	if err != nil {
		return nil, err
	}
	if f.username != "" {
		credsEncoded := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", f.username, f.password)))
		req.Header.Add("Authorization", fmt.Sprintf("Basic %s", credsEncoded))
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}
