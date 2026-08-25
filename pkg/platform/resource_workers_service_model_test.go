// Copyright (c) JFrog Ltd. (2025)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package platform

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func TestArtifactFilterCriteriaAPIModel_repoKeysJSON(t *testing.T) {
	t.Parallel()

	empty := []string{}
	tests := []struct {
		name string
		in   artifactFilterCriteriaAPIModel
		want string
	}{
		{
			name: "nil repoKeys omitted",
			in: artifactFilterCriteriaAPIModel{
				AnyLocal: boolPtr(true),
			},
			want: `{"anyLocal":true}`,
		},
		{
			name: "empty repoKeys transmitted",
			in: artifactFilterCriteriaAPIModel{
				RepoKeys: &empty,
			},
			want: `{"repoKeys":[]}`,
		},
		{
			name: "populated repoKeys transmitted",
			in: artifactFilterCriteriaAPIModel{
				RepoKeys: stringSlicePtr([]string{"a"}),
			},
			want: `{"repoKeys":["a"]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := json.Marshal(tt.in)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Fatalf("json.Marshal() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestWorkersServiceResourceModel_artifactFilterCriteriaRoundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sourceCode := "export default async () => ({ status: 'DOWNLOAD_PROCEED' })"

	tests := []struct {
		name       string
		artifact   artifactFilterCriteriaResourceModel
		wantRepo   types.Set
		wantAnyLoc *bool
	}{
		{
			name: "repo_keys with value",
			artifact: artifactFilterCriteriaResourceModel{
				RepoKeys: mustStringSet(t, ctx, "a"),
			},
			wantRepo: mustStringSet(t, ctx, "a"),
		},
		{
			name: "explicit empty repo_keys",
			artifact: artifactFilterCriteriaResourceModel{
				RepoKeys: mustStringSet(t, ctx),
			},
			wantRepo: mustStringSet(t, ctx),
		},
		{
			name: "omitted repo_keys with any_local",
			artifact: artifactFilterCriteriaResourceModel{
				RepoKeys: types.SetNull(types.StringType),
				AnyLocal: types.BoolValue(true),
			},
			wantRepo:   types.SetNull(types.StringType),
			wantAnyLoc: boolPtr(true),
		},
		{
			name: "omitted repo_keys without wildcards",
			artifact: artifactFilterCriteriaResourceModel{
				RepoKeys: types.SetNull(types.StringType),
			},
			wantRepo: types.SetNull(types.StringType),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			filterCriteria, d := types.ObjectValueFrom(ctx, filterCriteriaResourceModelAttributeTypes, filterCriteriaResourceModel{
				ArtifactFilterCriteria: mustArtifactFilterCriteriaObject(t, ctx, tt.artifact),
				Schedule:               types.ObjectNull(scheduleResourceModelAttributeTypes),
			})
			if len(d) > 0 {
				t.Fatalf("ObjectValueFrom() diagnostics = %v", d)
			}

			model := workersServiceResourceModel{
				Key:            types.StringValue("worker-key"),
				Description:    types.StringValue("desc"),
				SourceCode:     types.StringValue(sourceCode),
				Action:         types.StringValue("BEFORE_DOWNLOAD"),
				Enabled:        types.BoolValue(true),
				FilterCriteria: filterCriteria,
				Secrets:        types.SetNull(types.ObjectType{AttrTypes: map[string]attr.Type{"key": types.StringType, "value": types.StringType}}),
			}

			var apiModel WorkersServiceAPIModel
			ds := model.toAPIModel(ctx, &apiModel, nil)
			if ds.HasError() {
				t.Fatalf("toAPIModel() diagnostics = %v", ds)
			}

			var roundTrip workersServiceResourceModel
			ds = roundTrip.fromAPIModel(ctx, &apiModel)
			if ds.HasError() {
				t.Fatalf("fromAPIModel() diagnostics = %v", ds)
			}

			var gotFilter filterCriteriaResourceModel
			ds = roundTrip.FilterCriteria.As(ctx, &gotFilter, basetypes.ObjectAsOptions{
				UnhandledNullAsEmpty: true,
			})
			if ds.HasError() {
				t.Fatalf("FilterCriteria.As() diagnostics = %v", ds)
			}

			var gotArtifact artifactFilterCriteriaResourceModel
			ds = gotFilter.ArtifactFilterCriteria.As(ctx, &gotArtifact, basetypes.ObjectAsOptions{})
			if ds.HasError() {
				t.Fatalf("ArtifactFilterCriteria.As() diagnostics = %v", ds)
			}

			if !tt.wantRepo.Equal(gotArtifact.RepoKeys) {
				t.Fatalf("repo_keys: want %v got %v", tt.wantRepo, gotArtifact.RepoKeys)
			}

			if tt.wantAnyLoc == nil {
				if !gotArtifact.AnyLocal.IsNull() {
					t.Fatalf("any_local: want null got %v", gotArtifact.AnyLocal)
				}
			} else if gotArtifact.AnyLocal.ValueBool() != *tt.wantAnyLoc {
				t.Fatalf("any_local: want %v got %v", *tt.wantAnyLoc, gotArtifact.AnyLocal.ValueBool())
			}
		})
	}
}

func TestOptionalSlicePtrToStringSet(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	empty := []string{"x"}

	tests := []struct {
		name    string
		in      *[]string
		wantLen int
		wantNil bool
	}{
		{name: "nil pointer", in: nil, wantNil: true},
		{name: "empty slice", in: &[]string{}, wantLen: 0},
		{name: "values", in: &empty, wantLen: 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, ds := optionalSlicePtrToStringSet(ctx, tt.in)
			if ds.HasError() {
				t.Fatalf("optionalSlicePtrToStringSet() diagnostics = %v", ds)
			}

			if tt.wantNil {
				if !got.IsNull() {
					t.Fatalf("got set = %v, want null", got)
				}
				return
			}

			if got.IsNull() {
				t.Fatalf("got null set, want len %d", tt.wantLen)
			}
			if len(got.Elements()) != tt.wantLen {
				t.Fatalf("len(got.Elements()) = %d, want %d", len(got.Elements()), tt.wantLen)
			}
		})
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func stringSlicePtr(v []string) *[]string {
	return &v
}

func mustStringSet(t *testing.T, ctx context.Context, values ...string) types.Set {
	t.Helper()

	set, d := types.SetValueFrom(ctx, types.StringType, values)
	if len(d) > 0 {
		t.Fatalf("SetValueFrom() diagnostics = %v", d)
	}
	return set
}

func artifactFilterCriteriaWithDefaults(model artifactFilterCriteriaResourceModel) artifactFilterCriteriaResourceModel {
	if model.RepoKeys.IsNull() || model.RepoKeys.IsUnknown() {
		model.RepoKeys = types.SetNull(types.StringType)
	}
	if model.IncludePatterns.IsNull() || model.IncludePatterns.IsUnknown() {
		model.IncludePatterns = types.SetNull(types.StringType)
	}
	if model.ExcludePatterns.IsNull() || model.ExcludePatterns.IsUnknown() {
		model.ExcludePatterns = types.SetNull(types.StringType)
	}
	if model.AnyLocal.IsNull() || model.AnyLocal.IsUnknown() {
		model.AnyLocal = types.BoolNull()
	}
	if model.AnyRemote.IsNull() || model.AnyRemote.IsUnknown() {
		model.AnyRemote = types.BoolNull()
	}
	if model.AnyFederated.IsNull() || model.AnyFederated.IsUnknown() {
		model.AnyFederated = types.BoolNull()
	}
	return model
}

func mustArtifactFilterCriteriaObject(t *testing.T, ctx context.Context, model artifactFilterCriteriaResourceModel) types.Object {
	t.Helper()

	obj, d := types.ObjectValueFrom(ctx, artifactFilterCriteriaResourceModelAttributeTypes, artifactFilterCriteriaWithDefaults(model))
	if len(d) > 0 {
		t.Fatalf("ObjectValueFrom() diagnostics = %v", d)
	}
	return obj
}
