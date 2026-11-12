package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/strcut_json"
)

func TestNewSettings(t *testing.T) {

	inSettings := Settings{
		UserInfo: &UserInfo{
			Username: "abcd",
			Password: "123456",
		},
		CommonSettings: &CommonSettings{
			ScanInterval:     "12h",
			Threads:          12,
			RunScanAtStartUp: true,
			MoviePaths:       []string{"aaa", "bbb"},
			SeriesPaths:      []string{"ccc", "ddd"},
		},
		AdvancedSettings: &AdvancedSettings{
			ProxySettings: &ProxySettings{
				UseProxy:                 true,
				LocalHttpProxyServerPort: "123",
			},
			DebugMode:                  true,
			SaveFullSeasonTmpSubtitles: true,
			SubTypePriority:            1,
			SubNameFormatter:           1,
			SaveMultiSub:               true,
			CustomVideoExts:            []string{"aaa", "bbb"},
			FixTimeLine:                true,
		},
		EmbySettings: &EmbySettings{
			Enable:                true,
			AddressUrl:            "123456",
			APIKey:                "api123",
			MaxRequestVideoNumber: 1000,
			SkipWatched:           true,
			MoviePathsMapping:     map[string]string{"aa": "123", "bb": "456"},
			SeriesPathsMapping:    map[string]string{"aab": "123", "bbc": "456"},
		},
		DeveloperSettings: &DeveloperSettings{
			BarkServerAddress: "bark",
		},
	}

	fileName := filepath.Join(t.TempDir(), "testfile.json")
	err := strcut_json.ToFile(fileName, inSettings)
	if err != nil {
		t.Fatal(err)
	}

	outSettings := NewSettings("")
	err = strcut_json.ToStruct(fileName, &outSettings)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(inSettings.UserInfo, outSettings.UserInfo) == false {
		t.Fatal("inSettings Write And Read Not The Same")
	}
}

// A settings request (or an old config file) with sections missing or null must not leave
// nil pointers behind, and a save that fails must not replace the settings in memory.
func TestSetFullNewSettingsFillsMissingSections(t *testing.T) {
	dir := t.TempDir()
	SetConfigRootPath(dir)
	Get()

	in := &Settings{
		UserInfo:         &UserInfo{Username: "abcd", Password: "123456"},
		AdvancedSettings: &AdvancedSettings{Topic: 2},
		EmbySettings:     &EmbySettings{AddressUrl: "http://emby:8096//"},
	}
	if err := SetFullNewSettings(in); err != nil {
		t.Fatal(err)
	}
	got := Get()
	if got.CommonSettings == nil || got.SubtitleSources == nil || got.ExperimentalFunction == nil ||
		got.AdvancedSettings.ProxySettings == nil || got.AdvancedSettings.SuppliersSettings == nil ||
		got.AdvancedSettings.TaskQueue == nil || got.AdvancedSettings.SubPriority == nil ||
		got.AdvancedSettings.SubPriority.MatchRelease == nil || got.EmbySettings.MoviePathsMapping == nil {
		t.Fatalf("nil section left after SetFullNewSettings: %+v", got)
	}
	if got.AdvancedSettings.Topic != 2 || got.UserInfo.Username != "abcd" || got.EmbySettings.AddressUrl != "http://emby:8096" {
		t.Fatal("values sent in the request were not kept")
	}
	got.Check() // used to panic on nil TaskQueue

	// a save that fails keeps the old settings
	bad := NewSettings("")
	bad.UserInfo.Username = "other"
	bad.EmbySettings.AddressUrl = "http://[::1"
	if err := SetFullNewSettings(bad); err == nil {
		t.Fatal("want error for an unparsable media server url")
	}
	if Get().UserInfo.Username != "abcd" {
		t.Fatal("failed save replaced the settings in memory")
	}

	// config file written before sub_priority / server_type existed, or with null sections
	old := `{"user_info":{"username":"abcd","password":"123456"},"advanced_settings":{"sub_priority":null,"task_queue":null},"emby_settings":{"enable":true,"address_url":"http://emby:8096"},"common_settings":null}`
	if err := os.WriteFile(filepath.Join(dir, configName), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	re := Get().GetNoPasswordSettings()
	if re.CommonSettings == nil || re.AdvancedSettings.SubPriority == nil || re.AdvancedSettings.TaskQueue == nil {
		t.Fatal("nil section left after reading an old config")
	}
	if re.EmbySettings.IsJellyfin() || !re.EmbySettings.Enable {
		t.Fatal("old config without server_type must stay Emby and keep its values")
	}
	if len(re.AdvancedSettings.SubPriority.SiteOrder) != len(DefaultSiteOrder) {
		t.Fatal("sub_priority default not applied")
	}
}
