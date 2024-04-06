package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/plugins/kvstore"
)

type Site struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Url      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
	State    string `json:"state"`
}

type Twsnmp struct {
	Kvs   *kvstore.KeyValueStore
	Sites sync.Map
}

func (t *Twsnmp) GetVersion() string {
	return fmt.Sprintf("%s(%s)", version, commit)
}

func (t *Twsnmp) GetSites() []Site {
	ret := []Site{}
	t.Sites.Range(func(k, v any) bool {
		if s, ok := v.(Site); ok {
			ret = append(ret, s)
		}
		return true
	})
	return ret
}

func (t *Twsnmp) UpdateSite2(s Site) {
	if s.Id == "" {
		s.Id = fmt.Sprintf("%x", time.Now().UnixNano())
		s.State = "unknown"
	}
	t.Sites.Store(s.Id, s)
}

func (t *Twsnmp) UpdateSite(id, name, url, user, password string) {
	state := "unknow"
	if id == "" {
		id = fmt.Sprintf("%x", time.Now().UnixNano())
	}
	if v, ok := t.Sites.Load(id); ok {
		if s, ok := v.(Site); ok {
			state = s.State
		}
	}
	t.Sites.Store(id, Site{
		Id:       id,
		Name:     name,
		Url:      url,
		User:     user,
		Password: password,
		State:    state,
	})
}

// DeleteSite は Siteを削除します
func (t *Twsnmp) DeleteSite(id string) {
	t.Sites.Delete(id)
	t.Kvs.Set(id, nil)
}

func (t *Twsnmp) OpenSiteMap(id string) bool {
	v, ok := t.Sites.Load(id)
	if !ok {
		return false
	}
	s, ok := v.(Site)
	if !ok {
		return false
	}
	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title: "TWSNMP Map:" + s.Name,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
		},
		URL: "/?page=map&id=" + id,
	})

	return true
}

func (t *Twsnmp) load() {
	conf, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	t.Kvs = kvstore.NewPlugin(&kvstore.Config{
		Filename: filepath.Join(conf, "twsnmpmw.db"),
	})
	m, ok := t.Kvs.Get("").(map[string]any)
	if !ok {
		return
	}
	for k, v := range m {
		if strings.HasPrefix(k, "twsmp") {
			if j, ok := v.(string); ok {
				s := Site{}
				if err := json.Unmarshal([]byte(j), &s); err == nil {
					t.Sites.Store(s.Id, s)
				}
			}
		}
	}
}

func (t *Twsnmp) save() {
	t.Sites.Range(func(k, v any) bool {
		if id, ok := k.(string); ok {
			if s, ok := v.(Site); ok {
				k := "twsnmp_" + id
				if j, err := json.Marshal(&s); err == nil {
					if err := t.Kvs.Set(k, j); err != nil {
						log.Println(err)
					}
				}

			}
		}
		return true
	})
	t.Kvs.Save()
}

func (t *Twsnmp) checkSiteState() chan bool {
	ticker := time.NewTicker(60 * time.Second)
	stop := make(chan bool)
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				// Check site here
				log.Println("check site")
			}
		}
	}()
	return stop
}
