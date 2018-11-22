package core

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const genDirName = "./basenet"
const versionKey = "tm-version"
const pexKey = "pex"
const nodesKey = "nodes"
const mempoolSizeKey = "mempool-size"
const txCacheKeyKey = "tx-cache-size"
const p2pPortKey = "p2p-port"
const rpcPortKey = "rpc-port"
const proxyPortKey = "proxy-port"
const loggingPortKey = "logging-port"

type App struct {
	tmVersion   string
	pexEnabled  bool
	nodes       []string
	p2pPort     int
	rpcPort     int
	proxyPort   int
	loggingPort int
	txCacheSize int
	memPoolSize int
	config      *viper.Viper
	nodeConfigs []*viper.Viper
}

func New() *App {
	return &App{}
}

func (app *App) Run() {
	app.config = viper.New()

	app.setDefaults()
	app.updateConfig()
	err := app.readCustomConfig()
	if err != nil {
		log.Error(err.Error())
		return
	}
	app.updateConfig()
	app.reportCustomConfig()
	err = app.removeExisting()
	if err != nil {
		log.Error(err.Error())
		return
	}
	app.generateTestnet()
	if err != nil {
		log.Error(err.Error())
		return
	}
	err = app.readTestnetConfigs()
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Infof("done.")
}

func (app *App) setDefaults() {
	app.config.SetDefault(versionKey, "0.26.0")
	app.config.SetDefault(pexKey, false)
	app.config.SetDefault(nodesKey, []string{"192.168.0.1", "192.168.0.2", "192.168.0.3"})
	app.config.SetDefault(p2pPortKey, 26656)
	app.config.SetDefault(rpcPortKey, 26657)
	app.config.SetDefault(proxyPortKey, 26658)
	app.config.SetDefault(loggingPortKey, 26660)
	app.config.SetDefault(mempoolSizeKey, 5000)
	app.config.SetDefault(txCacheKeyKey, 10000)
}

func (app *App) updateConfig() {
	app.pexEnabled = app.config.GetBool(pexKey)
	app.nodes = app.config.GetStringSlice(nodesKey)
	app.tmVersion = app.config.GetString(versionKey)
	app.p2pPort = app.config.GetInt(p2pPortKey)
	app.rpcPort = app.config.GetInt(rpcPortKey)
	app.proxyPort = app.config.GetInt(proxyPortKey)
	app.loggingPort = app.config.GetInt(loggingPortKey)
	app.txCacheSize = app.config.GetInt(txCacheKeyKey)
	app.memPoolSize = app.config.GetInt(mempoolSizeKey)
}

func (app *App) readCustomConfig() error {
	log.Info("reading 'net.json' config files")
	app.config.SetConfigName("net")
	app.config.AddConfigPath(".")
	err := app.config.ReadInConfig()
	if err != nil {
		return fmt.Errorf("cannot load 'net.json' file: %s \n", err)
	}
	return nil
}

func (app *App) reportCustomConfig() {
	log.Infof(">> tendermint-version: %s", app.tmVersion)
	log.Infof(">> nodes: %v", app.nodes)
	log.Infof(">> nodes-total: %d", len(app.nodes))
	log.Infof(">> pexEnabled: %s", strconv.FormatBool(app.pexEnabled))
	log.Infof(">> p2p-port: %d", app.p2pPort)
	log.Infof(">> rpc-port: %d", app.rpcPort)
	log.Infof(">> proxy-port: %d", app.proxyPort)
	log.Infof(">> logging-port: %d", app.loggingPort)
	log.Infof(">> tx-cache-size: %d", app.txCacheSize)
	log.Infof(">> mempool-size: %d", app.memPoolSize)
}

func (app *App) removeExisting() error {
	log.Info("removing existing tendermint config files")
	return os.RemoveAll(genDirName)
}

func (app *App) generateTestnet() error {
	log.Info("generating tendermint config files")
	tmPath := fmt.Sprintf("./tendermint/%s/tendermint", app.tmVersion)
	log.Debugf(tmPath)
	output, err := exec.Command(tmPath, "testnet", "--o", genDirName, "--v", fmt.Sprintf("%d", len(app.nodes)), "--populate-persistent-peers", "--starting-ip-address", "192.168.0.1").CombinedOutput()
	if err != nil {
		return err
	}
	log.Debug(string(output))
	if output == nil {
		return fmt.Errorf("tendermint testnet output is nil")
	}
	if !strings.Contains(string(output), "Successfully") {
		return fmt.Errorf("tendermint testnet did not return success: %s", string(output))
	}
	return nil
}

func (app *App) readTestnetConfigs() error {
	log.Info("read tendermint generated config files")
	for i := 0; i < len(app.nodes); i++ {
		app.nodeConfigs = append(app.nodeConfigs, viper.New())

		log.Infof("reading node%d 'config.toml' file", i)

		app.nodeConfigs[i].SetConfigName("config")
		app.nodeConfigs[i].AddConfigPath(fmt.Sprintf("./basenet/node%d/config/", i))
		err := app.nodeConfigs[i].ReadInConfig()
		if err != nil {
			return fmt.Errorf("cannot load node%d 'config.toml' file: %s \n", i, err)
		}

		log.Infof("%v", app.nodeConfigs[i].AllKeys())
		peers := app.nodeConfigs[i].GetString("p2p.persistent_peers")

		app.nodeConfigs[i].Set("moniker", fmt.Sprintf("node%d", i))
		app.nodeConfigs[i].Set("proxy_app", fmt.Sprintf("tcp://127.0.0.1:%d", app.proxyPort))
		app.nodeConfigs[i].Set("p2p.laddr", fmt.Sprintf("tcp://0.0.0.0:%d", app.p2pPort))
		for j := 0; j < len(app.nodes); j++ {
			peers = strings.Replace(peers, fmt.Sprintf("192.168.0.%d:26656", j+1), fmt.Sprintf("%s:%d", app.nodes[j], app.p2pPort), 1)
		}
		log.Info(peers)
		app.nodeConfigs[i].Set("p2p.persistent_peers", peers)
		app.nodeConfigs[i].Set("p2p.cache_size", app.txCacheSize)
		app.nodeConfigs[i].Set("p2p.size", app.memPoolSize)
		app.nodeConfigs[i].Set("p2p.pex", strconv.FormatBool(app.pexEnabled))
		app.nodeConfigs[i].Set("instrumentation.prometheus_listen_addr", fmt.Sprintf(":%d", app.loggingPort))
		app.nodeConfigs[i].Set("rpc.laddr", fmt.Sprintf("tcp://0.0.0.0:%d", app.rpcPort))
		app.nodeConfigs[i].WriteConfig()

		log.Infof("updated tendermint generated config for node%d with custom properties", i)
	}

	return nil
}
