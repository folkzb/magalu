package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/sdk"
)

type ReadResult core.ResultWithValue

type MgcOperation interface {
	WrapConext(ctx context.Context) context.Context

	CollectParameters(ctx context.Context, state, plan TerraformParams) (core.Parameters, Diagnostics)
	CollectConfigs(ctx context.Context, state, plan TerraformParams) (core.Configs, Diagnostics)

	// If 'ShouldRun' returns false, 'Run' and 'PostRun' will be skipped. The chain operations will still be called, with
	// 'opResult' passed as nil
	ShouldRun(ctx context.Context, params core.Parameters, configs core.Configs) (run bool, d Diagnostics)
	Run(ctx context.Context, params core.Parameters, configs core.Configs) (core.ResultWithValue, Diagnostics)
	// If 'PostRun' returns false or an error, the chain operations will not be called
	// 'postResult' may be either the same result as 'Run' or something else. If 'postResult'
	// is equivalent to a read from the resource, the Runner won't re-read it, and will use
	// 'postResult' as 'readResult' in 'ChainOperations'
	PostRun(ctx context.Context, result core.ResultWithValue, state, plan TerraformParams, targetState *tfsdk.State) (postResult core.ResultWithValue, runChain bool, d Diagnostics)

	ChainOperations(ctx context.Context, opResult core.ResultWithValue, readResult ReadResult, state, plan TerraformParams) ([]MgcOperation, bool, Diagnostics)
}

type MgcReadOperation interface {
	MgcOperation
	ReadResultSchema() *mgcSchemaPkg.Schema
}

type mgcReadOperation struct {
	MgcOperation
	readResultSchema *mgcSchemaPkg.Schema
}

func (o *mgcReadOperation) ReadResultSchema() *mgcSchemaPkg.Schema {
	return o.readResultSchema
}

func wrapReadOperation(operation MgcOperation, readResultSchema *mgcSchemaPkg.Schema) MgcReadOperation {
	return &mgcReadOperation{operation, readResultSchema}
}

type MgcOperationRunner struct {
	sdk           *sdk.Sdk
	state         tfsdk.State
	plan          tfsdk.Plan
	targetState   *tfsdk.State
	rootOperation MgcOperation
	readOperation MgcReadOperation
}

func newMgcOperationRunner(
	sdk *sdk.Sdk,
	rootOperation MgcOperation,
	readOperation MgcReadOperation,
	state tfsdk.State,
	plan tfsdk.Plan,
	targetState *tfsdk.State,
) MgcOperationRunner {
	return MgcOperationRunner{
		sdk:           sdk,
		rootOperation: rootOperation,
		readOperation: readOperation,
		state:         state,
		plan:          plan,
		targetState:   targetState,
	}
}

func (r *MgcOperationRunner) Run(ctx context.Context) Diagnostics {
	ctx = r.sdk.WrapContext(ctx)
	diagnostics := Diagnostics{}

	opResult, opPostResult, runChain, d := r.runOperation(ctx, r.rootOperation)
	if diagnostics.AppendCheckError(d...) || !runChain {
		return diagnostics
	}

	state, plan, d := r.getCurrentTFParams()
	if diagnostics.AppendCheckError(d...) {
		return diagnostics
	}

	if r.readOperation == nil {
		return diagnostics
	}

	var readResult ReadResult
	if opPostResult != nil && r.readOperation.ReadResultSchema().VisitJSON(opPostResult.Value()) == nil {
		readResult = opPostResult
	} else if opResult != nil && r.readOperation.ReadResultSchema().VisitJSON(opResult.Value()) == nil {
		readResult = opResult
	} else {
		readResult, _, _, d = r.runOperation(ctx, r.readOperation)
		if diagnostics.AppendCheckError(d...) {
			return diagnostics
		}
	}

	chained, runChain, d := r.rootOperation.ChainOperations(ctx, opResult, readResult, state, plan)
	if diagnostics.AppendCheckError(d...) || !runChain {
		return diagnostics
	}

	r.runChainedOperations(ctx, chained)

	return diagnostics
}

func (r *MgcOperationRunner) getCurrentTFParams() (state, plan TerraformParams, d Diagnostics) {
	diagnostics := Diagnostics{}

	state, err := tfStateToParams(r.state)
	if err != nil {
		diagnostics.AddError(
			"error when reading Terraform state",
			fmt.Sprintf("Terraform state wasn't able to be read: %s", err.Error()),
		)
	}

	plan, err = tfStateToParams(tfsdk.State(r.plan))
	if err != nil {
		diagnostics.AddError(
			"error when reading Terraform state",
			fmt.Sprintf("Terraform plan wasn't able to be read: %s", err.Error()),
		)
	}

	return state, plan, diagnostics
}

func (r *MgcOperationRunner) runOperation(ctx context.Context, operation MgcOperation) (runResult core.ResultWithValue, postResult core.ResultWithValue, runChain bool, diagnostics Diagnostics) {
	ctx = operation.WrapConext(ctx)
	diagnostics = Diagnostics{}

	state, plan, d := r.getCurrentTFParams()
	if diagnostics.AppendCheckError(d...) {
		return nil, nil, false, diagnostics
	}

	params, d := operation.CollectParameters(ctx, state, plan)
	if diagnostics.AppendCheckError(d...) {
		return nil, nil, false, diagnostics
	}

	configs, d := operation.CollectConfigs(ctx, state, plan)
	if diagnostics.AppendCheckError(d...) {
		return nil, nil, false, diagnostics
	}

	shouldRun, d := operation.ShouldRun(ctx, params, configs)
	if diagnostics.AppendCheckError(d...) || !shouldRun {
		return nil, nil, true, diagnostics
	}

	runResult, d = operation.Run(ctx, params, configs)
	if diagnostics.AppendCheckError(d...) {
		return runResult, nil, true, diagnostics
	}

	d = validateResult(runResult)
	if diagnostics.AppendCheckError(d...) {
		return runResult, nil, false, diagnostics
	}

	postResult, runChain, d = operation.PostRun(ctx, runResult, state, plan, r.targetState)
	if diagnostics.AppendCheckError(d...) {
		return runResult, postResult, false, diagnostics
	}

	r.state = *r.targetState

	return runResult, postResult, runChain, diagnostics
}

func (r *MgcOperationRunner) runChainedOperations(ctx context.Context, chained []MgcOperation) Diagnostics {
	diagnostics := Diagnostics{}

	for _, operation := range chained {
		opResult, opPostResult, run, d := r.runOperation(ctx, operation)
		if diagnostics.AppendCheckError(d...) {
			return diagnostics
		}

		if !run {
			continue
		}

		if r.readOperation == nil {
			continue
		}

		var readResult ReadResult
		if opPostResult != nil && r.readOperation.ReadResultSchema().VisitJSON(opPostResult.Value()) == nil {
			readResult = opPostResult
		} else if opResult != nil && r.readOperation.ReadResultSchema().VisitJSON(opResult.Value()) == nil {
			readResult = opResult
		} else {
			readResult, _, _, d = r.runOperation(ctx, r.readOperation)
			if diagnostics.AppendCheckError(d...) {
				return diagnostics
			}
		}

		state, plan, d := r.getCurrentTFParams()
		if diagnostics.AppendCheckError(d...) {
			return diagnostics
		}

		chained, run, d := operation.ChainOperations(ctx, opResult, readResult, state, plan)
		if diagnostics.AppendCheckError(d...) {
			return diagnostics
		}

		if !run {
			continue
		}

		r.runChainedOperations(ctx, chained)
	}

	return diagnostics
}
