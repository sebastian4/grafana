package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/grafana/grafana-plugin-sdk-go/data"

	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/datasources"
	apimodels "github.com/grafana/grafana/pkg/services/ngalert/api/tooling/definitions"
	"github.com/grafana/grafana/pkg/services/ngalert/eval"
	ngmodels "github.com/grafana/grafana/pkg/services/ngalert/models"
	"github.com/grafana/grafana/pkg/util"
)

type TestingApiSrv struct {
	*AlertingProxy
	DatasourceCache datasources.CacheService
	log             log.Logger
	accessControl   accesscontrol.AccessControl
	evaluator       eval.Evaluator
}

func (srv TestingApiSrv) RouteTestGrafanaRuleConfig(c *models.ReqContext, body apimodels.TestRulePayload) response.Response {
	if body.Type() != apimodels.GrafanaBackend || body.GrafanaManagedCondition == nil {
		return errorToResponse(backendTypeDoesNotMatchPayloadTypeError(apimodels.GrafanaBackend, body.Type().String()))
	}

	if !authorizeDatasourceAccessForRule(&ngmodels.AlertRule{Data: body.GrafanaManagedCondition.Data}, func(evaluator accesscontrol.Evaluator) bool {
		return accesscontrol.HasAccess(srv.accessControl, c)(accesscontrol.ReqSignedIn, evaluator)
	}) {
		return errorToResponse(fmt.Errorf("%w to query one or many data sources used by the rule", ErrAuthorization))
	}

	evalCond := ngmodels.Condition{
		Condition: body.GrafanaManagedCondition.Condition,
		OrgID:     c.SignedInUser.OrgId,
		Data:      body.GrafanaManagedCondition.Data,
	}

	if err := validateCondition(c.Req.Context(), evalCond, c.SignedInUser, c.SkipCache, srv.DatasourceCache); err != nil {
		return ErrResp(http.StatusBadRequest, err, "invalid condition")
	}

	now := body.GrafanaManagedCondition.Now
	if now.IsZero() {
		now = timeNow()
	}

	evalResults := srv.evaluator.ConditionEval(c.Req.Context(), evalCond, now)

	frame := evalResults.AsDataFrame()
	return response.JSONStreaming(http.StatusOK, util.DynMap{
		"instances": []*data.Frame{&frame},
	})
}

func (srv TestingApiSrv) RouteTestRuleConfig(c *models.ReqContext, body apimodels.TestRulePayload, datasourceUID string) response.Response {
	if body.Type() != apimodels.LoTexRulerBackend {
		return errorToResponse(backendTypeDoesNotMatchPayloadTypeError(apimodels.LoTexRulerBackend, body.Type().String()))
	}
	ds, err := getDatasourceByUID(c, srv.DatasourceCache, apimodels.LoTexRulerBackend)
	if err != nil {
		return errorToResponse(err)
	}
	var path string
	switch ds.Type {
	case "loki":
		path = "loki/api/v1/query"
	case "prometheus":
		path = "api/v1/query"
	default:
		// this should not happen because getDatasourceByUID would not return the data source
		return errorToResponse(unexpectedDatasourceTypeError(ds.Type, "loki, prometheus"))
	}

	t := timeNow()
	queryURL, err := url.Parse(path)
	if err != nil {
		return ErrResp(http.StatusInternalServerError, err, "failed to parse url")
	}
	params := queryURL.Query()
	params.Set("query", body.Expr)
	params.Set("time", strconv.FormatInt(t.Unix(), 10))
	queryURL.RawQuery = params.Encode()
	return srv.withReq(
		c,
		http.MethodGet,
		queryURL,
		nil,
		instantQueryResultsExtractor,
		nil,
	)
}

func (srv TestingApiSrv) RouteEvalQueries(c *models.ReqContext, cmd apimodels.EvalQueriesPayload) response.Response {
	now := cmd.Now
	if now.IsZero() {
		now = timeNow()
	}

	if !authorizeDatasourceAccessForRule(&ngmodels.AlertRule{Data: cmd.Data}, func(evaluator accesscontrol.Evaluator) bool {
		return accesscontrol.HasAccess(srv.accessControl, c)(accesscontrol.ReqSignedIn, evaluator)
	}) {
		return ErrResp(http.StatusUnauthorized, fmt.Errorf("%w to query one or many data sources used by the rule", ErrAuthorization), "")
	}

	if _, err := validateQueriesAndExpressions(c.Req.Context(), cmd.Data, c.SignedInUser, c.SkipCache, srv.DatasourceCache); err != nil {
		return ErrResp(http.StatusBadRequest, err, "invalid queries or expressions")
	}

	evalResults, err := srv.evaluator.QueriesAndExpressionsEval(c.Req.Context(), c.SignedInUser.OrgId, cmd.Data, now)
	if err != nil {
		return ErrResp(http.StatusBadRequest, err, "Failed to evaluate queries and expressions")
	}

	return response.JSONStreaming(http.StatusOK, evalResults)
}
