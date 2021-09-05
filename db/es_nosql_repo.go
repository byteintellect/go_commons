package db

import (
	"github.com/byteintellect/go_commons"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/gobeam/stringy"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"reflect"
	"strings"
)

var (
	kindStr = map[reflect.Kind]string{
		reflect.String:  "text",
		reflect.Int64:   "long",
		reflect.Int32:   "integer",
		reflect.Bool:    "bool",
		reflect.Uint64:  "long",
		reflect.Slice:   "nested",
		reflect.Struct:  "nested",
		reflect.Float64: "float",
		reflect.Float32: "float",
		reflect.Int:     "integer",
	}
)

type FieldAnalysis struct {
	Analyzer       string
	SearchAnalyzer string
}

func (f *FieldAnalysis) String() string {
	return fmt.Sprintf("{\"analyzer\":\"%v\", \"search_analyzer\":\"%v\"}", f.Analyzer, f.SearchAnalyzer)
}

type Tokenizer string

type CharFilter string

type TokenFilter string

var TokenizerMapping = struct {
	Standard    Tokenizer
	Letter      Tokenizer
	Lowercase   Tokenizer
	Whitespace  Tokenizer
	Classic     Tokenizer
	Punctuation Tokenizer
}{
	Standard:    "standard",
	Letter:      "letter",
	Lowercase:   "lowercase",
	Whitespace:  "whitespace",
	Classic:     "classic",
	Punctuation: "punctuation",
}

var CharFilterMapping = struct {
	HTMLStrip     CharFilter
	Mapping       CharFilter
	PatterReplace CharFilter
}{
	HTMLStrip: "html_strip",
}

var TokenFilterMapping = struct {
	Lowercase    TokenFilter
	StopWords    TokenFilter
	AsciiFolding TokenFilter
}{
	Lowercase:    "lowercase",
	StopWords:    "english_stop",
	AsciiFolding: "asciifolding",
}

type ESHealth struct {
	ClusterName                 string  `json:"cluster_name"`
	Status                      string  `json:"status"`
	TimedOut                    bool    `json:"timed_out"`
	NumberOfNodes               uint    `json:"number_of_nodes"`
	ActivePrimaryShards         uint    `json:"active_primary_shards"`
	ActiveShards                uint    `json:"active_shards"`
	InitializingShards          uint    `json:"initializing_shards"`
	UnAssignedShards            uint    `json:"unassigned_shards"`
	DelayedUnassignedShards     uint    `json:"delayed_unassigned_shards"`
	NumberOfPendingTasks        uint    `json:"number_of_pending_tasks"`
	NumberOfInFlightFetch       uint    `json:"number_of_in_flight_fetch"`
	TaskMaxWaitingInQueueMillis uint    `json:"task_max_waiting_in_queue_millis"`
	ActiveShardsPercentAsNumber float32 `json:"active_shards_percent_as_number"`
}

type Shards struct {
	Total      uint `json:"total"`
	Successful uint `json:"successful"`
	Skipped    uint `json:"skipped"`
	Failed     uint `json:"failed"`
}

type Total struct {
	Value    uint   `json:"value"`
	Relation string `json:"relation"`
}

type EsHit struct {
	Index  string                 `json:"_index"`
	Type   string                 `json:"_type"`
	Id     string                 `json:"_id"`
	Score  float64                `json:"_score"`
	Source map[string]interface{} `json:"_source"`
}

type Hits struct {
	Total    Total   `json:"total"`
	MaxScore float64 `json:"max_score"`
	Hits     []EsHit `json:"hits"`
}

type ESSearchResponse struct {
	Took     uint   `json:"took"`
	TimedOut bool   `json:"timed_out"`
	Shards   Shards `json:"_shards"`
	Hits     Hits   `json:"hits"`
}

type ESQuery struct {
	AttrName string
	Value    interface{}
}

func (eq *ESQuery) Term() string {
	return ""
}

type HttpStatusChecker struct {
}

func (hsc *HttpStatusChecker) IsSuccessFul(statusCode int) bool {
	return statusCode/100 == 2
}

func (hsc *HttpStatusChecker) IsInternalError(statusCode int) bool {
	return statusCode/100 == 5
}

func (hsc *HttpStatusChecker) IsClientError(statusCode int) bool {
	return statusCode/100 == 4
}

type HttpBodyUtil struct {
}

func (hbu *HttpBodyUtil) BytesToResponse(resBody io.ReadCloser, factory func() interface{}) error {
	if err := json.NewDecoder(resBody).Decode(factory()); err != nil {
		return err
	}
	return nil
}

func (hbu *HttpBodyUtil) ResponseToString(factory func() interface{}) (string, error) {
	rBytes, err := json.Marshal(factory())
	if err != nil {
		return "", err
	}
	return string(rBytes), nil
}

type Analyzer struct {
	Type      string   `json:"type,omitempty"`
	Tokenizer string   `json:"tokenizer,omitempty"`
	Filter    []string `json:"filter,omitempty"`
}

type Analysis struct {
	Analyzer map[string]Analyzer `json:"analyzer,omitempty"`
	Filter   Filter              `json:"filter,omitempty"`
}

type FilterObj struct {
	Type      string `json:"type,omitempty"`
	StopWords string `json:"stop_words,omitempty"`
}

type Filter struct {
	Filter map[string]FilterObj `json:"filter,omitempty"`
}

type Settings struct {
	Analysis Analysis `json:"analysis"`
}

func getTagValue(rawTag, key string) string {
	tags := strings.Split(rawTag, ",")
	for _, tag := range tags {
		if strings.HasPrefix(tag, key) {
			return tag[len(key)+1:]
		}
	}
	return ""
}

func (esr *ElasticsearchRepo) getMappingForSlice(w reflect.Type, parentPath string) string {
	var mappings []string
	for j := 0; j < w.NumField(); j++ {
		if w.Field(j).Type.Kind() != reflect.Struct && w.Field(j).Type.Kind() != reflect.Slice && w.Field(j).Type.Kind() != reflect.Chan {
			if w.Field(j).Type.Kind() == reflect.String {
				mappings = append(mappings, fmt.Sprintf("   \"%v\": {\n     \"type\": \"%v\"\n, \"fields\": {\n\"keyword\": {\n \"type\": \"keyword\"} } }", toSnakeCase(w.Field(j).Name), kindStr[w.Field(j).Type.Kind()]))
				attrName := fmt.Sprintf("%v.%v", parentPath, toSnakeCase(w.Field(j).Name))
				esr.fieldMappings[attrName] = FieldAnalysis{
					Analyzer:       w.Field(j).Tag.Get("analyzer"),
					SearchAnalyzer: w.Field(j).Tag.Get("search_analyzer"),
				}
			} else {
				if w.Field(j).Type.Kind() == reflect.Ptr {
					mappings = append(mappings, fmt.Sprintf("\"%v\": {\"type\": \"%v\"}", toSnakeCase(w.Field(j).Name), w.Field(j).Tag.Get("type")))
				} else if w.Field(j).Tag.Get("type") != "" {
					mappings = append(mappings, fmt.Sprintf("\"%v\": {\"type\": \"%v\"}", toSnakeCase(w.Field(j).Name), w.Field(j).Tag.Get("type")))
				} else {
					mappings = append(mappings, fmt.Sprintf("\"%v\": {\"type\": \"%v\"}", toSnakeCase(w.Field(j).Name), kindStr[w.Field(j).Type.Kind()]))
				}
			}
		} else if w.Field(j).Type.Kind() == reflect.Struct || (w.Field(j).Type.Kind() == reflect.Slice && w.Field(j).Type.Elem().Kind() == reflect.Struct) {
			if w.Field(j).Tag.Get("type") != "" {
				mappings = append(mappings, fmt.Sprintf("\"%v\": {\"type\": \"%v\"}", toSnakeCase(w.Field(j).Name), w.Field(j).Tag.Get("type")))
			} else {
				mappings = append(mappings, fmt.Sprintf("\"%v\":{\"properties\": {\n %v \n}\n}", toSnakeCase(w.Field(j).Name), esr.getMappingForSlice(w.Field(j).Type, fmt.Sprintf("%v.%v", parentPath, toSnakeCase(w.Field(j).Name)))))
			}
		} else if w.Field(j).Type.Kind() == reflect.Slice && (w.Field(j).Type.Elem().Kind() != reflect.Struct && w.Field(j).Type.Elem().Kind() != reflect.Chan) {
			if w.Field(j).Type.Kind() == reflect.String {
				attrName := fmt.Sprintf("%v.%v", parentPath, toSnakeCase(w.Field(j).Name))
				esr.fieldMappings[attrName] = FieldAnalysis{
					Analyzer:       w.Field(j).Tag.Get("analyzer"),
					SearchAnalyzer: w.Field(j).Tag.Get("search_analyzer"),
				}
				mappings = append(mappings, fmt.Sprintf("   \"%v\": {\n \"type\": \"text\"\n }", toSnakeCase(w.Field(j).Name)))
			} else {
				attrName := fmt.Sprintf("%v.%v", parentPath, toSnakeCase(w.Field(j).Name))
				esr.fieldMappings[attrName] = FieldAnalysis{
					Analyzer:       w.Field(j).Tag.Get("analyzer"),
					SearchAnalyzer: w.Field(j).Tag.Get("search_analyzer"),
				}
				mappings = append(mappings, fmt.Sprintf("\"%v\": {\"type\": \"%v\"}", toSnakeCase(w.Field(j).Name), w.Field(j).Tag.Get("type")))
			}
		}
	}
	return strings.Join(mappings, ",\n")
}

func (esr *ElasticsearchRepo) getMapping(v reflect.Value) string {
	var mappings []string
	valType := v.Type()
	for i := 0; i < v.NumField(); i++ {
		attr := v.Field(i)
		if attr.Kind() != reflect.Struct && attr.Kind() != reflect.Slice && attr.Kind() != reflect.Chan {
			if attr.Kind() == reflect.String {
				attrName := toSnakeCase(valType.Field(i).Name)
				esr.fieldMappings[attrName] = FieldAnalysis{
					Analyzer:       valType.Field(i).Tag.Get("analyzer"),
					SearchAnalyzer: valType.Field(i).Tag.Get("search_analyzer"),
				}
				mappings = append(mappings, fmt.Sprintf("   \"%v\": {\n     \"type\": \"%v\"\n, \"fields\": {\n\"keyword\": {\n \"type\": \"keyword\"} } }", toSnakeCase(valType.Field(i).Name), kindStr[attr.Kind()]))
			} else {
				mappings = append(mappings, fmt.Sprintf("   \"%v\": {\n     \"type\": \"%v\"\n   }", toSnakeCase(valType.Field(i).Name), kindStr[attr.Kind()]))
			}
		} else if attr.Kind() == reflect.Slice {
			attrType := reflect.TypeOf(attr.Interface()).Elem().Kind()
			if attrType != reflect.Struct {
				if attrType == reflect.String {
					attrName := toSnakeCase(valType.Field(i).Name)
					mappings = append(mappings, fmt.Sprintf("   \"%v\": {\n     \"type\": \"text\"\n }", toSnakeCase(valType.Field(i).Name)))
					esr.fieldMappings[attrName] = FieldAnalysis{
						Analyzer:       valType.Field(i).Tag.Get("analyzer"),
						SearchAnalyzer: valType.Field(i).Tag.Get("search_analyzer"),
					}
				} else {
					mappings = append(mappings, fmt.Sprintf("   \"%v\": {\n     \"type\": \"%v\"\n   }", toSnakeCase(valType.Field(i).Name), kindStr[attrType]))
				}
			} else {
				sFace := reflect.TypeOf(attr.Interface()).Elem()
				mappings = append(mappings, fmt.Sprintf("\"%v\":{\n    \"properties\":    {\n %v }\n  }", toSnakeCase(valType.Field(i).Name), esr.getMappingForSlice(sFace, toSnakeCase(valType.Field(i).Name))))
			}
		} else if attr.Kind() == reflect.Struct {
			mappings = append(mappings, fmt.Sprintf("\"%v\":{\n    \"properties\":    {\n %v }\n  }", toSnakeCase(valType.Field(i).Name), esr.getMappingForSlice(reflect.TypeOf(attr.Interface()), toSnakeCase(valType.Field(i).Name))))
		}
	}
	return "{\n \"mappings\":{ \n  \"properties\":{\n" + strings.Join(mappings, ",\n") + "\n  }\n }\n}"
}

type ElasticsearchRepo struct {
	marshaller      *HttpBodyUtil
	client          *elasticsearch.Client
	sChecker        *HttpStatusChecker
	entityCreator   go_commons.EntityCreator
	index           string
	entityConverter func(from map[string]interface{}) go_commons.Base
	fieldMappings   map[string]FieldAnalysis
	defaultEntity   go_commons.Base
	logger          *logrus.Logger
	settings        Settings
	httpClient      *http.Client
}

type ElasticsearchRepoOption func(repo *ElasticsearchRepo)

func WithHttpClient(client *http.Client) ElasticsearchRepoOption {
	return func(repo *ElasticsearchRepo) {
		repo.httpClient = client
	}
}

func WithMarshaller(marshaller *HttpBodyUtil) ElasticsearchRepoOption {
	return func(repo *ElasticsearchRepo) {
		repo.marshaller = marshaller
	}
}

func WithIndex(index string) ElasticsearchRepoOption {
	return func(repo *ElasticsearchRepo) {
		repo.index = index
	}
}

func WithESLogger(logger *logrus.Logger) ElasticsearchRepoOption {
	return func(repo *ElasticsearchRepo) {
		repo.logger = logger
	}
}

func WithClient(client *elasticsearch.Client) ElasticsearchRepoOption {
	return func(repo *ElasticsearchRepo) {
		repo.client = client
	}
}

func WithSettings(settings Settings) ElasticsearchRepoOption {
	return func(repo *ElasticsearchRepo) {
		repo.settings = settings
	}
}

func WithDefaultEntity(base go_commons.Base) ElasticsearchRepoOption {
	return func(repo *ElasticsearchRepo) {
		repo.defaultEntity = base
	}
}

func WithEntityConverter(converter func(from map[string]interface{}) go_commons.Base) ElasticsearchRepoOption {
	return func(repo *ElasticsearchRepo) {
		repo.entityConverter = converter
	}
}

func WithEntityCreator(creator go_commons.EntityCreator) ElasticsearchRepoOption {
	return func(repo *ElasticsearchRepo) {
		repo.entityCreator = creator
	}
}

func WithStatusChecker(checker *HttpStatusChecker) ElasticsearchRepoOption {
	return func(repo *ElasticsearchRepo) {
		repo.sChecker = checker
	}
}

func NewElasticsearchRepo(opts ...ElasticsearchRepoOption) BaseNoSQLRepo {
	repo := &ElasticsearchRepo{
		fieldMappings: make(map[string]FieldAnalysis),
	}

	for _, opt := range opts {
		opt(repo)
	}
	return repo
}

func (esr *ElasticsearchRepo) GetById(ctx context.Context, id uint64) (error, go_commons.Base) {
	req := esapi.SearchRequest{
		Index: []string{esr.index},
		Body:  strings.NewReader(fmt.Sprintf("{\"query\":{\"term\":{\"id\":%v}}}", id)),
	}
	res, err := req.Do(ctx, esr.client)
	if err != nil || !esr.sChecker.IsSuccessFul(res.StatusCode) {
		return err, nil
	}
	var response ESSearchResponse
	err = esr.marshaller.BytesToResponse(res.Body, func() interface{} {
		return &response
	})
	if err != nil {
		return err, nil
	}
	for _, hit := range response.Hits.Hits {
		return nil, esr.entityConverter(hit.Source)
	}
	return errors.New("not found"), nil
}

func (esr *ElasticsearchRepo) Search(ctx context.Context, params map[string]string) (error, []go_commons.Base) {
	var queries []string
	for key, value := range params {
		// supports filtering only on text fields
		queries = append(queries, fmt.Sprintf("{\"term\": {\"%v\": \"%v\"}\n}", key, value))
	}
	req := esapi.SearchRequest{
		Index: []string{esr.index},
		Body:  strings.NewReader(fmt.Sprintf("{\"query\":{\"bool\":{\"filter\":[%v]\n}\n}\n}", strings.Join(queries, ",\n"))),
	}
	res, err := req.Do(ctx, esr.client)
	if err != nil || !esr.sChecker.IsSuccessFul(res.StatusCode) {
		return err, nil
	}
	var response ESSearchResponse
	err = esr.marshaller.BytesToResponse(res.Body, func() interface{} {
		return &response
	})
	if err != nil {
		return err, nil
	}
	var result []go_commons.Base
	for _, hit := range response.Hits.Hits {
		result = append(result, esr.entityConverter(hit.Source))
	}
	return nil, result
}

func (esr *ElasticsearchRepo) GetDb() interface{} {
	return esr.client
}

func (esr *ElasticsearchRepo) ExactSearch(ctx context.Context, key string, value interface{}) (error, []go_commons.Base) {
	req := esapi.SearchRequest{
		Index: []string{esr.index},
		Body:  strings.NewReader(fmt.Sprintf("{\"query\":{\"term\":{\"%v\":%v}}}", key, value)),
	}
	res, err := req.Do(ctx, esr.client)
	if err != nil || !esr.sChecker.IsSuccessFul(res.StatusCode) {
		return err, nil
	}
	var response ESSearchResponse
	err = esr.marshaller.BytesToResponse(res.Body, func() interface{} {
		return &response
	})
	if err != nil {
		return err, nil
	}
	var result []go_commons.Base
	for _, hit := range response.Hits.Hits {
		result = append(result, esr.entityConverter(hit.Source))
	}
	return nil, result
}

func (esr *ElasticsearchRepo) RangeSearch(ctx context.Context, key string, start, end interface{}) (error, []go_commons.Base) {
	queryString := fmt.Sprintf("{\"query\":{\"range\": {\"%v\": {\"gte\": %v, \"lte\": %v}}}}", key, start, end)
	req := esapi.SearchRequest{
		Index: []string{esr.index},
		Body:  strings.NewReader(queryString),
	}

	res, err := req.Do(ctx, esr.client)
	if err != nil || !esr.sChecker.IsSuccessFul(res.StatusCode) {
		return err, nil
	}
	var response ESSearchResponse
	err = esr.marshaller.BytesToResponse(res.Body, func() interface{} {
		return &response
	})
	if err != nil {
		return err, nil
	}
	var result []go_commons.Base
	for _, hit := range response.Hits.Hits {
		result = append(result, esr.entityConverter(hit.Source))
	}
	return nil, result
}

func (esr *ElasticsearchRepo) getMatchQuery(value string) []string {
	var result []string
	for attr, mapping := range esr.fieldMappings {
		var analyzer string
		if mapping.Analyzer != "" {
			analyzer = fmt.Sprintf(", \"analyzer\": %v", mapping.Analyzer)
		}
		result = append(result, fmt.Sprintf("{\"query\": {\"match\": {\"%v\": {\"query\": %v %v}}}}", attr, value, analyzer))
	}
	return result
}

func (esr *ElasticsearchRepo) getMatchPhraseQuery(value string) []string {
	var result []string
	for attr, mapping := range esr.fieldMappings {
		var analyzer string
		if mapping.Analyzer != "" {
			analyzer = fmt.Sprintf(", \"analyzer\": %v", mapping.Analyzer)
		}
		result = append(result, fmt.Sprintf("{\"query\": {\"match_phrase\": {\"%v\": {\"query\": %v %v}}}}", attr, value, analyzer))
	}
	return result
}

func (esr *ElasticsearchRepo) getMultiMatchQuery(queryType, value string) string {
	var attrs []string
	var analyzerType string
	for attr, mapping := range esr.fieldMappings {
		if mapping.Analyzer != "" {
			analyzerType = fmt.Sprintf(", \"analyzer\": \"%v\"", mapping.Analyzer)
		}
		attrs = append(attrs, fmt.Sprintf("\"%v\"", attr))
	}
	return fmt.Sprintf("{ \"multi_match\": {\"query\": \"%v\", \"type\": \"%v\", \"fields\": [%v] %v}}", value, queryType, strings.Join(attrs, ","), analyzerType)
}

func (esr *ElasticsearchRepo) TextSearch(ctx context.Context, value string) (error, []go_commons.Base) {
	var innerQueries []string
	innerQueries = append(innerQueries, esr.getMultiMatchQuery("cross_fields", value))
	innerQueries = append(innerQueries, esr.getMultiMatchQuery("best_fields", value))
	innerQueries = append(innerQueries, esr.getMultiMatchQuery("phrase", value))
	innerQueries = append(innerQueries, esr.getMultiMatchQuery("phrase_prefix", value))
	queryBody := fmt.Sprintf("{\"query\": {\"bool\":{\"should\":[%v]}}}", strings.Join(innerQueries, ","))
	req := esapi.SearchRequest{
		Index: []string{esr.index},
		Body:  strings.NewReader(queryBody),
	}
	res, err := req.Do(ctx, esr.client)
	if err != nil || !esr.sChecker.IsSuccessFul(res.StatusCode) {
		return err, nil
	}
	var response ESSearchResponse
	err = esr.marshaller.BytesToResponse(res.Body, func() interface{} {
		return &response
	})
	if err != nil {
		return err, nil
	}
	var result []go_commons.Base
	for _, hit := range response.Hits.Hits {
		result = append(result, esr.entityConverter(hit.Source))
	}
	return nil, result
}

func (esr *ElasticsearchRepo) IndexMappings(ctx context.Context) error {
	v := reflect.ValueOf(esr.defaultEntity)
	var mapping map[string]interface{}
	err := json.Unmarshal([]byte(esr.getMapping(v)), &mapping)
	if err != nil {
		log.Fatalf("An error %v occurred while indexing mappings, cannot continue.", err)
	}
	mapping["settings"] = esr.settings
	mappingStr, err := json.Marshal(mapping)
	esr.logger.Infof("Mappings %v", string(mappingStr))
	if err != nil {
		log.Fatalf("An error %v occurred while marshalling mapping to json", err)
	}
	req, err := http.NewRequest(http.MethodPut, "http://localhost:9200/"+esr.index, bytes.NewBuffer(mappingStr))
	if err != nil {
		return err
	}
	req.Header.Add("content-type", "application/json")
	resp, err := esr.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	esr.logger.Infof("Mapping created %v", string(respBytes))
	return nil
}

func (esh *ESHealth) IsHealthy() bool {
	return (esh.Status == "yellow" || esh.Status == "green") && esh.ActiveShardsPercentAsNumber >= 50.00
}

func (esr *ElasticsearchRepo) IsHealthy() bool {
	info, err := esr.client.Cluster.Health()
	if err != nil {
		return false
	}
	var esResponse ESHealth
	err = esr.marshaller.BytesToResponse(info.Body, func() interface{} {
		return &esResponse
	})
	if err != nil {
		return false
	}
	return esResponse.IsHealthy()
}

func (esr *ElasticsearchRepo) Create(ctx context.Context, base go_commons.Base) (error, go_commons.Base) {
	jBody, err := base.ToJson()
	if err != nil {
		return err, nil
	}
	req := esapi.IndexRequest{
		Index:      string(base.GetTable()),
		DocumentID: base.GetExternalId(),
		Body:       strings.NewReader(jBody),
		Refresh:    "true",
	}
	res, err := req.Do(ctx, esr.client)
	if err != nil || esr.sChecker.IsInternalError(res.StatusCode) || esr.sChecker.IsClientError(res.StatusCode) {
		return errors.New(fmt.Sprintf("Error while indexing %v", err)), nil
	}

	return nil, base
}

func (esr *ElasticsearchRepo) Update(ctx context.Context, entityId string, base go_commons.Base) (error, go_commons.Base) {
	req := esapi.UpdateRequest{DocumentID: entityId, Index: base.GetExternalId()}
	res, err := req.Do(ctx, esr.client)
	if err != nil || esr.sChecker.IsInternalError(res.StatusCode) || esr.sChecker.IsClientError(res.StatusCode) {
		return errors.New(fmt.Sprintf("Error while updating %v", err)), nil
	}
	return nil, base
}

func (esr *ElasticsearchRepo) GetByExternalId(ctx context.Context, entityId string) (error, go_commons.Base) {
	truthy := true
	req := esapi.GetRequest{DocumentID: entityId, Refresh: &truthy, Realtime: &truthy}
	res, err := req.Do(ctx, esr.client)
	if err != nil {
		return err, nil
	}
	entity := esr.entityCreator()
	err = esr.marshaller.BytesToResponse(res.Body, func() interface{} {
		return &entity
	})
	if err != nil {
		return err, nil
	}
	return nil, entity
}

func toSnakeCase(input string) string {
	return stringy.New(input).SnakeCase().ToLower()
}

func (esr *ElasticsearchRepo) MultiGetByExternalId(ctx context.Context, entityIds []string) (error, []go_commons.Base) {
	var nEntityIds []string
	for _, entityId := range entityIds {
		nEntityIds = append(nEntityIds, fmt.Sprintf("\"%v\"", entityId))
	}
	req := esapi.SearchRequest{
		Index: []string{esr.index},
		Body:  strings.NewReader(fmt.Sprintf("{\"query\":{\"terms\":{\"_id\":[%v]}}}", strings.Join(nEntityIds, ","))),
	}
	res, err := req.Do(ctx, esr.client)
	if err != nil || !esr.sChecker.IsSuccessFul(res.StatusCode) {
		return err, nil
	}
	var response ESSearchResponse
	err = esr.marshaller.BytesToResponse(res.Body, func() interface{} {
		return &response
	})
	if err != nil {
		return err, nil
	}
	var result []go_commons.Base
	for _, hit := range response.Hits.Hits {
		result = append(result, esr.entityConverter(hit.Source))
	}
	return nil, result
}
