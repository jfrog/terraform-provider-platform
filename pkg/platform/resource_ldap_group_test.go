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

package platform_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/jfrog/terraform-provider-platform/v2/pkg/platform"
)

// The LDAP group import needs a real LDAP server, so these tests run against a fake JFrog.

const (
	ldapGroupSetting = "TestLDAPGroups"
	ldapGroupBaseDN  = "ou=Groups,dc=example,dc=com"
)

type fakeLDAPGroup struct {
	realm string
	attrs string
}

type fakeJFrog struct {
	mu            sync.Mutex
	server        *httptest.Server
	groups        map[string]fakeLDAPGroup
	settingStatus int
	importStatus  int
	imports       []map[string]any
	deletes       int
	missingReads  int
	version       string
}

func newFakeJFrog(t *testing.T) *fakeJFrog {
	t.Helper()
	f := &fakeJFrog{groups: map[string]fakeLDAPGroup{}, settingStatus: http.StatusOK, importStatus: http.StatusNoContent, version: "7.100.0"}
	f.server = httptest.NewServer(http.HandlerFunc(f.handle))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeJFrog) put(name, dn string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.groups[name] = fakeLDAPGroup{realm: "ldap", attrs: fakeAttrs(name, dn)}
}

func (f *fakeJFrog) remove(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.groups, name)
}

func (f *fakeJFrog) setRealm(name, realm string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	g := f.groups[name]
	g.realm = realm
	f.groups[name] = g
}

func (f *fakeJFrog) importCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.imports)
}

func fakeAttrs(name, dn string) string {
	return fmt.Sprintf("ldapGroupName=%s;groupsStrategy=DYNAMIC;groupDn=%s", name, dn)
}

func (f *fakeJFrog) handle(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	const groupPath = "/artifactory/api/security/groups/"
	const settingPath = "/access/api/v1/ldap/groups/"
	switch {
	case r.URL.Path == settingPath+"import" && r.Method == http.MethodPost:
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if f.importStatus >= 300 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(f.importStatus)
			_, _ = w.Write([]byte(`{"errors":[{"code":"FORBIDDEN","message":"import not allowed"}]}`))
			return
		}
		f.imports = append(f.imports, body)
		for _, g := range body["import_groups"].([]any) {
			item := g.(map[string]any)
			name, dn := item["group_name"].(string), item["group_dn"].(string)
			f.groups[name] = fakeLDAPGroup{realm: "ldap", attrs: fakeAttrs(name, dn)}
		}
		w.WriteHeader(f.importStatus)
	case strings.HasPrefix(r.URL.Path, settingPath) && r.Method == http.MethodGet:
		if f.settingStatus != http.StatusOK {
			w.WriteHeader(f.settingStatus)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":                   strings.TrimPrefix(r.URL.Path, settingPath),
			"enabled_ldap":           "test",
			"group_base_dn":          ldapGroupBaseDN,
			"group_name_attribute":   "cn",
			"group_member_attribute": "member",
			"sub_tree":               false,
			"force_attribute_search": false,
			"filter":                 "(objectClass=groupOfNames)",
			"description_attribute":  "description",
			"strategy":               "DYNAMIC",
		})
	case strings.HasPrefix(r.URL.Path, groupPath) && r.Method == http.MethodGet:
		name := strings.TrimPrefix(r.URL.Path, groupPath)
		g, ok := f.groups[name]
		if ok && f.missingReads > 0 {
			f.missingReads--
			ok = false
		}
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/vnd.org.jfrog.artifactory.security.Group+json")
		_ = json.NewEncoder(w).Encode(map[string]any{"name": name, "realm": g.realm, "realmAttributes": g.attrs})
	case strings.HasPrefix(r.URL.Path, groupPath) && r.Method == http.MethodDelete:
		name := strings.TrimPrefix(r.URL.Path, groupPath)
		if _, ok := f.groups[name]; !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		delete(f.groups, name)
		f.deletes++
	case r.URL.Path == "/artifactory/api/system/version":
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"version":%q}`, f.version)
	default:
		// usage calls made by the provider
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeJFrog) checkCalls(imports, deletes int) func(*terraform.State) error {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if len(f.imports) != imports || f.deletes != deletes {
			return fmt.Errorf("expected %d imports and %d deletes, got %d and %d", imports, deletes, len(f.imports), f.deletes)
		}
		return nil
	}
}

func (f *fakeJFrog) checkNoGroups(*terraform.State) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.groups) != 0 {
		return fmt.Errorf("groups left in JFrog: %v", f.groups)
	}
	return nil
}

func ldapGroupConfig(f *fakeJFrog, name, dn string) string {
	return ldapGroupConfigWithSetting(f, name, dn, ldapGroupSetting)
}

func ldapGroupConfigWithSetting(f *fakeJFrog, name, dn, setting string) string {
	return fmt.Sprintf(`
provider "platform" {
	url          = %q
	access_token = "test-token"
}

resource "platform_ldap_group" "test" {
	name               = %q
	group_dn           = %q
	ldap_group_setting = %q
}`, f.server.URL, name, dn, setting)
}

func ldapDN(cn string) string {
	return fmt.Sprintf("cn=%s,%s", cn, ldapGroupBaseDN)
}

func TestLDAPGroup_lifecycle(t *testing.T) {
	f := newFakeJFrog(t)
	name := "my-ldap-group"
	const fqrn = "platform_ldap_group.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		CheckDestroy:             f.checkNoGroups,
		Steps: []resource.TestStep{
			{
				Config: ldapGroupConfig(f, name, ldapDN("My-LDAP-Group")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "id", name),
					resource.TestCheckResourceAttr(fqrn, "name", name),
					resource.TestCheckResourceAttr(fqrn, "group_dn", ldapDN("My-LDAP-Group")),
					resource.TestCheckResourceAttr(fqrn, "ldap_group_setting", ldapGroupSetting),
					f.checkCalls(1, 0),
				),
			},
			{
				ResourceName:      fqrn,
				ImportState:       true,
				ImportStateId:     ldapGroupSetting + "/" + name,
				ImportStateVerify: true,
			},
			{
				// A different DN replaces the group.
				Config: ldapGroupConfig(f, name, ldapDN("Other-Group")),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "group_dn", ldapDN("Other-Group")),
					f.checkCalls(2, 1),
				),
			},
		},
	})
}

func TestLDAPGroup_settingChangeIsInPlace(t *testing.T) {
	f := newFakeJFrog(t)
	const fqrn = "platform_ldap_group.test"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{Config: ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group"))},
			{
				Config: ldapGroupConfigWithSetting(f, "my-ldap-group", ldapDN("My-LDAP-Group"), "OtherSetting"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fqrn, "ldap_group_setting", "OtherSetting"),
					f.checkCalls(1, 0),
				),
			},
		},
	})
}

func TestLDAPGroup_dnSpellingFromJFrogIsNotDrift(t *testing.T) {
	f := newFakeJFrog(t)
	cfg := ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group"))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{Config: cfg},
			{
				PreConfig: func() { f.put("my-ldap-group", strings.ToUpper(ldapDN("My-LDAP-Group"))) },
				Config:    cfg,
				PlanOnly:  true,
			},
		},
	})
}

func TestLDAPGroup_failedReadbackIsReplacedOnNextApply(t *testing.T) {
	defer platform.ShortenLDAPGroupReadback()()
	f := newFakeJFrog(t)
	// Covers every read in the wait loop, so the group never shows up.
	f.missingReads = 4
	cfg := ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group"))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		CheckDestroy:             f.checkNoGroups,
		Steps: []resource.TestStep{
			{Config: cfg, ExpectError: regexp.MustCompile(`not visible`)},
			{Config: cfg, Check: f.checkCalls(2, 1)},
		},
	})
}

func TestLDAPGroup_importSendsSettingAndGroup(t *testing.T) {
	f := newFakeJFrog(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{{
			Config: ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group")),
			Check: func(*terraform.State) error {
				f.mu.Lock()
				defer f.mu.Unlock()
				if len(f.imports) != 1 {
					return fmt.Errorf("expected one import call, got %d", len(f.imports))
				}
				body := f.imports[0]
				setting := body["ldap_group_settings"].(map[string]any)
				if setting["name"] != ldapGroupSetting || setting["group_base_dn"] != ldapGroupBaseDN || setting["strategy"] != "DYNAMIC" {
					return fmt.Errorf("unexpected setting in the import call: %v", setting)
				}
				group := body["import_groups"].([]any)[0].(map[string]any)
				if group["group_name"] != "my-ldap-group" || group["group_dn"] != ldapDN("My-LDAP-Group") || group["required_update"] != "DOES_NOT_EXIST" {
					return fmt.Errorf("unexpected group in the import call: %v", group)
				}
				return nil
			},
		}},
	})
}

func TestLDAPGroup_createRefusesExistingGroup(t *testing.T) {
	f := newFakeJFrog(t)
	f.put("my-ldap-group", ldapDN("My-LDAP-Group"))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{{
			Config:      ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group")),
			ExpectError: regexp.MustCompile(`already exists`),
		}},
	})
	if f.importCount() != 0 {
		t.Fatalf("expected no import call, got %d", f.importCount())
	}
}

func TestLDAPGroup_missingSettingSendsNoImport(t *testing.T) {
	f := newFakeJFrog(t)
	f.settingStatus = http.StatusNotFound

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{{
			Config:      ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group")),
			ExpectError: regexp.MustCompile(`LDAP group setting`),
		}},
	})
	if f.importCount() != 0 {
		t.Fatalf("expected no import call, got %d", f.importCount())
	}
}

func TestLDAPGroup_rejectedImportFails(t *testing.T) {
	f := newFakeJFrog(t)
	f.importStatus = http.StatusForbidden

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{{
			Config:      ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group")),
			ExpectError: regexp.MustCompile(`403.*import not allowed`),
		}},
	})
}

func TestLDAPGroup_readbackLagIsRetried(t *testing.T) {
	defer platform.ShortenLDAPGroupReadback()()
	f := newFakeJFrog(t)
	// The first read after the import finds nothing, as if the group had not replicated yet.
	f.missingReads = 1

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{{
			Config: ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group")),
			Check:  resource.TestCheckResourceAttr("platform_ldap_group.test", "id", "my-ldap-group"),
		}},
	})
}

func TestLDAPGroup_groupDeletedOutsideTerraformIsPlanned(t *testing.T) {
	f := newFakeJFrog(t)
	cfg := ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group"))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{Config: cfg},
			{
				PreConfig:          func() { f.remove("my-ldap-group") },
				Config:             cfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestLDAPGroup_otherRealmFailsRefresh(t *testing.T) {
	f := newFakeJFrog(t)
	cfg := ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group"))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{Config: cfg},
			{
				PreConfig:   func() { f.setRealm("my-ldap-group", "internal") },
				Config:      cfg,
				ExpectError: regexp.MustCompile(`realm`),
			},
			{
				PreConfig: func() { f.setRealm("my-ldap-group", "ldap") },
				Config:    cfg,
				PlanOnly:  true,
			},
		},
	})
}

func TestLDAPGroup_changedDNInJFrogIsDrift(t *testing.T) {
	f := newFakeJFrog(t)
	cfg := ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group"))

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{Config: cfg},
			{
				PreConfig:          func() { f.put("my-ldap-group", ldapDN("OTHER")) },
				Config:             cfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestLDAPGroup_oldArtifactoryVersionIsRejected(t *testing.T) {
	f := newFakeJFrog(t)
	f.version = "7.56.0"

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{{
			Config:      ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group")),
			ExpectError: regexp.MustCompile(`7\.57\.1`),
		}},
	})
	if f.importCount() != 0 {
		t.Fatalf("expected no import call, got %d", f.importCount())
	}
}

func TestLDAPGroup_invalidImportIDFails(t *testing.T) {
	f := newFakeJFrog(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProviders(),
		Steps: []resource.TestStep{
			{
				Config:        ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group")),
				ResourceName:  "platform_ldap_group.test",
				ImportState:   true,
				ImportStateId: "my-ldap-group",
				ExpectError:   regexp.MustCompile(`<ldap_group_setting>/<name>`),
			},
			{
				Config:        ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group")),
				ResourceName:  "platform_ldap_group.test",
				ImportState:   true,
				ImportStateId: "/my-ldap-group",
				ExpectError:   regexp.MustCompile(`<ldap_group_setting>/<name>`),
			},
			{
				Config:        ldapGroupConfig(f, "my-ldap-group", ldapDN("My-LDAP-Group")),
				ResourceName:  "platform_ldap_group.test",
				ImportState:   true,
				ImportStateId: ldapGroupSetting + "/",
				ExpectError:   regexp.MustCompile(`<ldap_group_setting>/<name>`),
			},
		},
	})
}
