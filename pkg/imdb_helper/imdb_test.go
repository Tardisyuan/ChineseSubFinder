package imdb_helper

import (
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/local_http_proxy_server"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/media_info_dealers"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/tmdb_api"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
	"testing"
)

func TestIsChineseVideo(t *testing.T) {
	type args struct {
		imdbID  string
		isMovie bool
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{name: "杀死伊芙", args: args{
			imdbID: "tt7016936", isMovie: false,
		}, want: false, wantErr: false,
		},
		{name: "云南虫谷", args: args{
			imdbID: "tt15299712", isMovie: false,
		}, want: true, wantErr: false,
		},
		{name: "扫黑风暴", args: args{
			imdbID: "tt15199554", isMovie: true,
		}, want: true, wantErr: false,
		},
		{name: "倚天屠龙记", args: args{
			imdbID: "tt1471140", isMovie: false,
		}, want: true, wantErr: false,
		},
		{name: "Only Murders in the Building", args: args{
			imdbID: "tt12851524", isMovie: false,
		}, want: false, wantErr: false,
		},
	}

	// needs a TMDB API key in ./ChineseSubFinderSettings.json besides the network
	unit_test_helper.NeedNetwork(t)
	defInstance()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, imdbInfo, err := IsChineseVideo(dealers, types.VideoNfoInfo{ImdbId: tt.args.imdbID})
			if (err != nil) != tt.wantErr {
				t.Errorf("IsChineseVideo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsChineseVideo() got = %v, want %v", got, tt.want)
			}
			println("IsChineseVideo:", got)
			println(imdbInfo.Name)
			println(imdbInfo.Year)
			println(imdbInfo.Languages[0])
		})
	}
}

func defInstance() {

	settings.SetConfigRootPath(".")
	err := local_http_proxy_server.SetProxyInfo(settings.Get().AdvancedSettings.ProxySettings.GetInfos())
	if err != nil {
		panic(err)
	}
	dealers = media_info_dealers.NewDealers(log_helper.GetLogger4Tester())

	tmdbApi, err := tmdb_api.NewTmdbHelper(log_helper.GetLogger4Tester(), settings.Get().AdvancedSettings.TmdbApiSettings.ApiKey, true)
	if err != nil {
		panic(err)
	}
	dealers.SetTmdbHelperInstance(tmdbApi)
}

var (
	dealers *media_info_dealers.Dealers
)
