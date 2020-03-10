// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	jsonutil "k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"net/url"
	"path"
)

func (p *Provider) bulk(data []byte) error {
	body, err := p.request(http.MethodPost, "_bulk", bytes.NewBuffer(data))
	if err != nil {
		return err
	}

	bulkRes := &BulkResponse{}
	if err := jsonutil.Unmarshal(body, bulkRes); err != nil {
		return errors.Wrap(err, "unable to unmarshal bulk response")
	}

	if bulkRes.Errors {
		items := make([]map[string]BulkResponseItem, 0)
		if err := jsonutil.Unmarshal(bulkRes.Items, &items); err != nil {
			return errors.Wrap(err, "unable to unmarshal bulk request items")
		}

		if len(bulkRes.Items) == 0 {
			return errors.New("elastic search returned an error")
		}
		var allErrors *multierror.Error
		for _, action := range items {
			for _, item := range action {
				if item.Status < 200 || item.Status > 299 {
					allErrors = multierror.Append(allErrors, errors.New(fmt.Sprintf("%#v", item.Error)))
				}
			}
		}
		return allErrors
	}

	return nil
}

func (p *Provider) request(httpMethod, rawPath string, payload io.Reader) ([]byte, error) {
	esURL, err := p.parseUrl(rawPath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(httpMethod, esURL, payload)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(p.config.Username, p.config.Password)
	req.Header.Add("Content-Type", "application/x-ndjson")
	req.Header.Add("Accept", "application/json")

	res, err := p.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to do request to %s", esURL)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode > 299 {
		statusErr := errors.Errorf("request %s returned status code %d", esURL, res.StatusCode)
		// also try to log the body is possible
		body, err := ioutil.ReadAll(res.Body)
		if err != nil {
			p.log.Error(err, "unable to read response body")
			return nil, statusErr
		}
		p.log.V(5).Info(statusErr.Error(), "body", string(body))
		return nil, statusErr
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read response body")
	}
	return body, err
}

func (p *Provider) parseUrl(rawPath string) (string, error) {
	u, err := url.Parse(p.config.Endpoint)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, rawPath)
	return u.String(), nil
}

// BulkResponse is the response that is returned by elastic search when doing a bulk request
type BulkResponse struct {
	Took   int             `json:"took"`
	Errors bool            `json:"errors"`
	Items  json.RawMessage `json:"items"` // []map[string]BulkResponseItem
}

// BulkResponseItem is response of one document from a bulk request
type BulkResponseItem struct {
	Index  string      `json:"_index"`
	Type   string      `json:"_type"`
	ID     string      `json:"_id"`
	Status int         `json:"status"`
	Error  interface{} `json:"error"`
}
