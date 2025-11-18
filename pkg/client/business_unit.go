package client

type BusinessUnitClient struct {
	httpClient *HttpClient
}

func NewBusinessUnitClient(baseUrl string) *BusinessUnitClient {
	return &BusinessUnitClient{
		httpClient: NewHttpClient(baseUrl),
	}
}
