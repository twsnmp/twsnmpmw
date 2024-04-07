package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"go.etcd.io/bbolt"
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
	db    *bbolt.DB
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
	s := Site{
		Id:       id,
		Name:     name,
		Url:      url,
		User:     user,
		Password: password,
		State:    state,
	}
	t.Sites.Store(id, s)
	t.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("twsnmp"))
		if j, err := json.Marshal(&s); err == nil {
			b.Put([]byte(id), j)
		} else {
			return err
		}
		return nil
	})

}

// DeleteSite は Siteを削除します
func (t *Twsnmp) DeleteSite(id string) {
	t.Sites.Delete(id)
	t.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("twsnmp"))
		return b.Delete([]byte(id))
	})
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
	os.MkdirAll(filepath.Join(conf, "twsnmpmw"), 0766)
	t.db, err = bbolt.Open(filepath.Join(conf, "twsnmpmw", "twsnmpmw.db"), 0600, nil)
	if err != nil {
		log.Fatalf("Open err=%v", err)
	}
	t.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("twsnmp"))
		if b == nil {
			b, err = tx.CreateBucketIfNotExists([]byte("twsnmp"))
			if err != nil {
				log.Fatal(err)
			}
		}
		b.ForEach(func(k, v []byte) error {
			id := string(k)
			var s Site
			if err := json.Unmarshal(v, &s); err == nil && id == s.Id {
				t.Sites.Store(id, s)
			}
			return nil
		})
		return nil
	})
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
