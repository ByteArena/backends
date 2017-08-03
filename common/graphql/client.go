package graphql

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Client struct {
	url string
}

func MakeClient(url string) Client {
	return Client{
		url: url,
	}
}

func NewClient(url string) *Client {
	return &Client{
		url: url,
	}
}

type graphqlwrapper struct {
	Data   json.RawMessage `json:"data"`
	Errors json.RawMessage `json:"errors"`
}

type Response struct {
	Body  json.RawMessage
	Error error
}

type Variables map[string]interface{}

type Query struct {
	query     string
	variables Variables
}

func NewQuery(query string) *Query {
	return &Query{
		query: query,
	}
}

func (q *Query) SetVariables(variables Variables) *Query {
	q.variables = variables
	return q
}

func (q *Query) HasVariables() bool {
	return q.variables != nil && len(q.variables) > 0
}

func (client Client) RequestSync(query *Query) (json.RawMessage, error) {
	future := client.RequestAsync(query)
	resp := <-future
	return resp.Body, resp.Error
}

func (client Client) RequestAsync(query *Query) <-chan Response {
	c := make(chan Response, 1)

	go func() {
		form := url.Values{}
		form.Add("query", query.query)

		if query.HasVariables() {
			jsonvariables, err := json.Marshal(query.variables)
			if err != nil {
				c <- Response{Error: err}
			}
			form.Add("variables", string(jsonvariables))
		}

		req, err := http.NewRequest("POST", client.url, strings.NewReader(form.Encode()))
		if err != nil {
			c <- Response{Error: err}
		}
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		httpclient := &http.Client{}
		resp, err := httpclient.Do(req)
		if err != nil {
			c <- Response{Error: err}
			return
		}

		// defered after err because if err, resp is nil
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			c <- Response{Error: err}
			return
		}

		var message graphqlwrapper

		err = json.Unmarshal(body, &message)
		if err != nil {
			c <- Response{Error: errors.New("Could not understand response of graphql server")}
			return
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			c <- Response{Error: errors.New("GraphQL server emitted an HTTP status code " + strconv.Itoa(resp.StatusCode) + "; " + string(message.Errors))}
			return
		}

		c <- Response{Body: message.Data}
	}()

	return c
}

func (client Client) Ping() error {
	resp, err := http.Get(client.url + "/schema")
	defer resp.Body.Close()

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return errors.New("HTTP error, status " + strconv.Itoa(resp.StatusCode))
	}

	return nil
}
