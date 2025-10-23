// Package prerequest provides pre-request plugins for GIE.
package prerequest

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/plugins"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/requestcontrol"
	"sigs.k8s.io/gateway-api-inference-extension/pkg/epp/scheduling/types"

	"github.com/llm-d/llm-d-inference-scheduler/pkg/common"
)

const (
	// PrefillHeaderHandlerType is the type of the PrefillHeaderHandler
	PrefillHeaderHandlerType = "prefill-header-handler"

	defaultPrefillProfile = "prefill"
)

type prefillHeaderHandlerParameters struct {
	PrefillProfile string `json:"prefillProfile"`
}

// compile-time type assertion
var _ requestcontrol.PreRequest = &PrefillHeaderHandler{}

// PrefillHeaderHandlerFactory  defines the factory function for the PrefillHeaderHandler
func PrefillHeaderHandlerFactory(name string, rawParameters json.RawMessage, _ plugins.Handle) (plugins.Plugin, error) {
	parameters := prefillHeaderHandlerParameters{
		PrefillProfile: defaultPrefillProfile,
	}
	if rawParameters != nil {
		if err := json.Unmarshal(rawParameters, &parameters); err != nil {
			return nil, fmt.Errorf("failed to parse the parameters of the '%s' pre-request plugin - %w", PrefillHeaderHandlerType, err)
		}
	}
	return NewPrefillHeaderHandler(parameters.PrefillProfile).WithName(name), nil
}

// NewPrefillHeaderHandler initializes a new PrefillHeaderHandler and returns its pointer.
func NewPrefillHeaderHandler(prefillProfile string) *PrefillHeaderHandler {
	return &PrefillHeaderHandler{
		typedName:      plugins.TypedName{Type: PrefillHeaderHandlerType},
		prefillProfile: prefillProfile,
	}
}

// PrefillHeaderHandler PreRequest plugin
type PrefillHeaderHandler struct {
	typedName      plugins.TypedName
	prefillProfile string
}

// TypedName returns the typed name of the plugin.
func (p *PrefillHeaderHandler) TypedName() plugins.TypedName {
	return p.typedName
}

// WithName sets the name of the plugin.
func (p *PrefillHeaderHandler) WithName(name string) *PrefillHeaderHandler {
	p.typedName.Name = name
	return p
}

// PreRequest wires prefill SchedulerProfile result into a header to indicate prefill worker
func (p *PrefillHeaderHandler) PreRequest(_ context.Context, request *types.LLMRequest, schedulingResult *types.SchedulingResult) {
	if _, found := request.Headers[common.PrefillPodHeader]; found {
		request.Headers[common.PrefillPodHeader] = "" // clear header, if already set
	}

	prefillProfileRunResult, exists := schedulingResult.ProfileResults[p.prefillProfile]
	if !exists {
		return // prefill profile failed to run or we chose not to run it, no-op in this case
	}

	targetPod := prefillProfileRunResult.TargetPods[0].GetPod()
	prefillHostPort := net.JoinHostPort(targetPod.Address, targetPod.Port)
	request.Headers[common.PrefillPodHeader] = prefillHostPort // in the form of <ip:port>
}
