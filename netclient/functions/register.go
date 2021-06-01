package functions

import (
	"time"
	"os"
	"log"
	"io/ioutil"
	"bytes"
        "github.com/gravitl/netmaker/netclient/config"
        "github.com/gravitl/netmaker/netclient/local"
        "github.com/gravitl/netmaker/netclient/wireguard"
        "github.com/gravitl/netmaker/models"
	"encoding/json"
	"net/http"
	"errors"
	"github.com/davecgh/go-spew/spew"
)

func Register(cfg config.GlobalConfig) error {

	_, err := os.Stat("/etc/netclient")
        if os.IsNotExist(err) {
                os.Mkdir("/etc/netclient", 744)
        } else if err != nil {
                log.Println("couldnt find or create /etc/netclient")
                return err
        }

        postclient := &models.IntClient{
                AccessKey: cfg.Client.AccessKey,
                PublicKey: cfg.Client.PublicKey,
                PrivateKey: cfg.Client.PublicKey,
		Address: cfg.Client.Address,
		Address6: cfg.Client.Address6,
		Network: "comms",
	}
	jsonstring, err := json.Marshal(postclient)
        if err != nil {
                return err
        }
	jsonbytes := []byte(jsonstring)
	body := bytes.NewBuffer(jsonbytes)
	log.Println("registering to http://"+cfg.Client.ServerAPIEndpoint+"/api/client/register")
	res, err := http.Post("http://"+cfg.Client.ServerEndpoint+"/api/intclient/register","application/json",body)
        if err != nil {
                return err
        }
	if res.StatusCode != http.StatusOK {
		return errors.New("request to server failed: " + res.Status)
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var wgclient models.IntClient
	json.Unmarshal(bodyBytes, &wgclient)
        spew.Dump(wgclient)
	err = config.ModGlobalConfig(wgclient)
        if err != nil {
                return err
        }

	err = wireguard.InitGRPCWireguard(wgclient)
        if err != nil {
                return err
        }

	return err
}

func Unregister(cfg config.GlobalConfig) error {
	client := &http.Client{ Timeout: 7 * time.Second,}
	req, err := http.NewRequest("DELETE", "http://"+cfg.Client.ServerAPIEndpoint+"/api/intclient/"+cfg.Client.ClientID, nil)
        if err != nil {
                return err
        }
	res, err := client.Do(req)
        if res == nil {
                return errors.New("server not reachable at " + "http://"+cfg.Client.ServerAPIEndpoint+"/api/intclient/"+cfg.Client.ClientID)

	} else if res.StatusCode != http.StatusOK {
                return errors.New("request to server failed: " + res.Status)
                defer res.Body.Close()
	} else {
	        err = local.WipeGRPCClient()
		if err == nil {
			log.Println("successfully removed grpc client interface")
		}
	}
	return err
}

func Reregister(cfg config.GlobalConfig) error {
	err := Unregister(cfg)
	if err != nil {
		log.Println("failed to un-register")
		return err
	}
	err = Register(cfg)
	if err != nil {
		log.Println("failed to re-register after unregistering")
	}
	return err
}
