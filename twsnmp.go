package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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
	Loc      string `json:"loc"`
	State    string `json:"state"`
}

type Api struct {
	Id    string
	Url   string
	Token string
	State string
}

type Twsnmp struct {
	db    *bbolt.DB
	Sites sync.Map
	api   sync.Map
}

func (t *Twsnmp) GetVersion() string {
	return fmt.Sprintf("%s(%s)", version, commit)
}

func (t *Twsnmp) GetSites() []Site {
	ret := []Site{}
	t.Sites.Range(func(k, v any) bool {
		if s, ok := v.(*Site); ok {
			if av, ok := t.api.Load(s.Id); ok {
				if a, ok := av.(*Api); ok {
					s.State = a.State
				}
			}
			ret = append(ret, *s)
		}
		return true
	})
	return ret
}

func (t *Twsnmp) UpdateSite(id, name, url, user, password string) {
	state := "unknow"
	if id == "" {
		id = fmt.Sprintf("%x", time.Now().UnixNano())
	}
	if v, ok := t.Sites.Load(id); ok {
		if s, ok := v.(*Site); ok {
			state = s.State
		}
	}
	s := &Site{
		Id:       id,
		Name:     name,
		Url:      url,
		User:     user,
		Password: password,
		State:    state,
	}
	t.Sites.Store(id, s)
	t.api.Delete(id)
	t.api.Store(id, &Api{
		Id:  id,
		Url: s.Url,
	})
	t.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("twsnmp"))
		if j, err := json.Marshal(s); err == nil {
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
	t.api.Delete(id)
	t.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("twsnmp"))
		return b.Delete([]byte(id))
	})
}

func (t *Twsnmp) UpdateSiteLoc(id, loc string) {
	v, ok := t.Sites.Load(id)
	if !ok {
		return
	}
	s, ok := v.(*Site)
	if !ok {
		return
	}
	s.Loc = loc
	t.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("twsnmp"))
		if j, err := json.Marshal(s); err == nil {
			b.Put([]byte(id), j)
		} else {
			return err
		}
		return nil
	})
}

func (t *Twsnmp) OpenSiteMap(id string) bool {
	v, ok := t.Sites.Load(id)
	if !ok {
		return false
	}
	s, ok := v.(*Site)
	if !ok {
		return false
	}
	app.NewWebviewWindowWithOptions(application.WebviewWindowOptions{
		Title: "TWSNMP Map:" + s.Name,
		Mac: application.MacWindow{
			Backdrop: application.MacBackdropTranslucent,
		},
		URL: s.Url + "/?user=" + s.User + "&password=" + s.Password,
	})
	return true
}

func (t *Twsnmp) LoadViewport() string {
	vp := ""
	t.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		if b != nil {
			if v := b.Get([]byte("viewport")); v != nil {
				vp = string(v)
			}
		}
		return nil
	})
	return vp
}

func (t *Twsnmp) SaveViewport(vp string) {
	t.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("config"))
		if err != nil {
			log.Println(err)
			return err
		}
		return b.Put([]byte("viewport"), []byte(vp))
	})
}
func (t *Twsnmp) IsDark() bool {
	r := false
	t.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		if b != nil {
			if v := b.Get([]byte("dark")); v != nil {
				r = string(v) == "true"
			}
		}
		return nil
	})
	return r
}

func (t *Twsnmp) SaveDark(r bool) {
	t.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("config"))
		if err != nil {
			log.Println(err)
			return err
		}
		v := "false"
		if r {
			v = "true"
		}
		return b.Put([]byte("dark"), []byte(v))
	})
}

func (t *Twsnmp) GetMapStyle() string {
	r := ""
	t.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("config"))
		if b != nil {
			if v := b.Get([]byte("mapStyle")); v != nil {
				r = string(v)
			}
		}
		return nil
	})
	return r
}

func (t *Twsnmp) SaveMapStyle(ms string) {
	t.db.Update(func(tx *bbolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("config"))
		if err != nil {
			log.Println(err)
			return err
		}
		return b.Put([]byte("mapStyle"), []byte(ms))
	})
}

// loadはsiteの設定を読み込む
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
				t.Sites.Store(id, &s)
				t.api.Store(id, &Api{
					Id:  id,
					Url: s.Url,
				})
			}
			return nil
		})
		return nil
	})
}

func (t *Twsnmp) checkSiteState() chan bool {
	ticker := time.NewTicker(10 * time.Second)
	stop := make(chan bool)
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				// Check site here
				log.Println("check site")
				t.Sites.Range(func(k, v any) bool {
					if s, ok := v.(*Site); ok {
						av, ok := t.api.Load(s.Id)
						if !ok {
							return true
						}
						a, ok := av.(*Api)
						if !ok {
							return true
						}
						if a.Token == "" {
							if !a.login(loginParam{UserID: s.User, Password: s.Password}) {
								a.State = "unknown"
								return true
							}
						}
						a.GetState()
					}
					return true
				})
			}
		}
	}()
	return stop
}

var insecureTransport = &http.Transport{
	TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
}

var insecureClient = &http.Client{Transport: insecureTransport}

type loginParam struct {
	UserID   string `json:"UserID"`
	Password string `json:"Password"`
}

type loginResp struct {
	Token string `json:"token"`
}

func (a *Api) login(l loginParam) bool {
	j, err := json.Marshal(&l)
	if err != nil {
		log.Println(err)
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequest(http.MethodPost, a.Url+"/login", bytes.NewBuffer(j))
	if err != nil {
		log.Println(err)
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := insecureClient.Do(req.WithContext(ctx))
	if err != nil {
		log.Println(err)
		return false
	}
	defer resp.Body.Close()
	if resp.ContentLength > 1024 {
		log.Printf("size over %d", resp.ContentLength)
		return false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return false
	}
	var r loginResp
	if err := json.Unmarshal(body, &r); err != nil {
		log.Println(err)
		return false
	}
	a.Token = r.Token
	return true
}

type nodeEnt struct {
	Name  string `json:"Name"`
	State string `json:"State"`
}

func (a *Api) GetState() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequest(http.MethodGet, a.Url+"/api/nodes", nil)
	if err != nil {
		log.Println(err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+a.Token)
	resp, err := insecureClient.Do(req.WithContext(ctx))
	if err != nil {
		log.Println(err)
		return
	}
	defer resp.Body.Close()
	if resp.ContentLength > 1024*64*1024 {
		log.Printf("size over %d", resp.ContentLength)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return
	}
	var nodes []nodeEnt
	if err := json.Unmarshal(body, &nodes); err != nil {
		log.Println(err)
		return
	}
	a.State = "unknown"
	for _, n := range nodes {
		switch n.State {
		case "high":
			a.State = "high"
			return
		case "low":
			a.State = "low"
		case "warn":
			if a.State != "low" {
				a.State = "warn"
			}
		case "normal", "repair":
			if a.State == "unknown" {
				a.State = "normal"
			}
		}
	}
}
