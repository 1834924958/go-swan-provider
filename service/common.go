package service

import (
	"fmt"
	"strings"
	"swan-provider/common/constants"
	"swan-provider/config"
	"time"

	"github.com/filswan/go-swan-lib/client"
	"github.com/filswan/go-swan-lib/logs"
	"github.com/filswan/go-swan-lib/model"
	"github.com/filswan/go-swan-lib/utils"

	"github.com/filswan/go-swan-lib/client/swan"
)

const ARIA2_TASK_STATUS_ERROR = "error"
const ARIA2_TASK_STATUS_ACTIVE = "active"
const ARIA2_TASK_STATUS_COMPLETE = "complete"

const DEAL_STATUS_CREATED = "Created"
const DEAL_STATUS_WAITING = "Waiting"

const DEAL_STATUS_DOWNLOADING = "Downloading"
const DEAL_STATUS_DOWNLOADED = "Downloaded"
const DEAL_STATUS_DOWNLOAD_FAILED = "DownloadFailed"

const DEAL_STATUS_IMPORT_READY = "ReadyForImport"
const DEAL_STATUS_IMPORTING = "FileImporting"
const DEAL_STATUS_IMPORTED = "FileImported"
const DEAL_STATUS_IMPORT_FAILED = "ImportFailed"
const DEAL_STATUS_ACTIVE = "DealActive"

const ONCHAIN_DEAL_STATUS_ERROR = "StorageDealError"
const ONCHAIN_DEAL_STATUS_ACTIVE = "StorageDealActive"
const ONCHAIN_DEAL_STATUS_NOTFOUND = "StorageDealNotFound"
const ONCHAIN_DEAL_STATUS_WAITTING = "StorageDealWaitingForData"
const ONCHAIN_DEAL_STATUS_ACCEPT = "StorageDealAcceptWait"
const ONCHAIN_DEAL_STATUS_AWAITING = "StorageDealAwaitingPreCommit"

const ARIA2_MAX_DOWNLOADING_TASKS = 10
const LOTUS_IMPORT_NUMNBER = "20" //Max number of deals to be imported at a time
const LOTUS_SCAN_NUMBER = "100"   //Max number of deals to be scanned at a time

var aria2Client *client.Aria2Client
var swanClient *swan.SwanClient

var swanService = GetSwanService()
var aria2Service = GetAria2Service()
var lotusService = GetLotusService()

func AdminOfflineDeal() {
	setAndCheckAria2Config()
	setAndCheckSwanConfig()
	checkMinerExists()
	checkLotusConfig()

	swanService.UpdateBidConf(swanClient)
	go swanSendHeartbeatRequest()
	go aria2CheckDownloadStatus()
	go aria2StartDownload()
	go lotusStartImport()
	go lotusStartScan()
}

func setAndCheckAria2Config() {
	aria2DownloadDir := config.GetConfig().Aria2.Aria2DownloadDir
	aria2Host := config.GetConfig().Aria2.Aria2Host
	aria2Port := config.GetConfig().Aria2.Aria2Port
	aria2Secret := config.GetConfig().Aria2.Aria2Secret

	if !utils.IsDirExists(aria2DownloadDir) {
		err := fmt.Errorf("aria2 down load dir:%s not exits, please set config:aria2->aria2_download_dir", aria2DownloadDir)
		logs.GetLogger().Fatal(err)
	}

	if len(aria2Host) == 0 {
		logs.GetLogger().Fatal("please set config:aria2->aria2_host")
	}

	aria2Client = client.GetAria2Client(aria2Host, aria2Secret, aria2Port)
}

func setAndCheckSwanConfig() {
	var err error
	swanApiUrl := config.GetConfig().Main.SwanApiUrl
	swanApiKey := config.GetConfig().Main.SwanApiKey
	swanAccessToken := config.GetConfig().Main.SwanAccessToken

	if len(swanApiUrl) == 0 {
		logs.GetLogger().Fatal("please set config:main->api_url")
	}

	if len(swanApiKey) == 0 {
		logs.GetLogger().Fatal("please set config:main->api_key")
	}

	if len(swanAccessToken) == 0 {
		logs.GetLogger().Fatal("please set config:main->access_token")
	}

	swanClient, err = swan.SwanGetClient(swanApiUrl, swanApiKey, swanAccessToken, "")
	if err != nil {
		logs.GetLogger().Error(err)
		logs.GetLogger().Error(constants.ERROR_LAUNCH_FAILED)
		logs.GetLogger().Fatal(constants.INFO_ON_HOW_TO_CONFIG)
	}
}

func checkMinerExists() {
	err := swanService.SendHeartbeatRequest(swanClient)
	if err != nil {
		logs.GetLogger().Info(err)
		if strings.Contains(err.Error(), "Miner Not found") {
			logs.GetLogger().Error("Cannot find your miner:", swanService.MinerFid)
		}
		logs.GetLogger().Error(constants.ERROR_LAUNCH_FAILED)
		logs.GetLogger().Fatal(constants.INFO_ON_HOW_TO_CONFIG)
	}
}

func checkLotusConfig() {
	logs.GetLogger().Info("Start testing lotus config.")

	if lotusService == nil {
		logs.GetLogger().Fatal("error in config")
	}

	lotusMarket := lotusService.LotusMarket
	lotusClient := lotusService.LotusClient
	if len(lotusMarket.ApiUrl) == 0 {
		logs.GetLogger().Fatal("please set config:lotus->market_api_url")
	}

	if len(lotusMarket.AccessToken) == 0 {
		logs.GetLogger().Fatal("please set config:lotus->market_access_token")
	}

	if len(lotusMarket.ClientApiUrl) == 0 {
		logs.GetLogger().Fatal("please set config:lotus->client_api_url")
	}

	err := lotusMarket.LotusImportData("bafyreib7azyg2yubucdhzn64gvyekdma7nbrbnfafcqvhsz2mcnvbnkktu", "test")

	if err != nil && !strings.Contains(err.Error(), "no such file or directory") && !strings.Contains(err.Error(), "datastore: key not found") {
		logs.GetLogger().Fatal(err)
	}

	currentEpoch := lotusClient.LotusGetCurrentEpoch()
	if currentEpoch < 0 {
		logs.GetLogger().Fatal("please check config:lotus->api_url")
	}

	logs.GetLogger().Info("Pass testing lotus config.")
}

func swanSendHeartbeatRequest() {
	for {
		logs.GetLogger().Info("Start...")
		swanService.SendHeartbeatRequest(swanClient)
		logs.GetLogger().Info("Sleeping...")
		time.Sleep(swanService.ApiHeartbeatInterval)
	}
}

func aria2CheckDownloadStatus() {
	for {
		logs.GetLogger().Info("Start...")
		aria2Service.CheckDownloadStatus(aria2Client, swanClient)
		logs.GetLogger().Info("Sleeping...")
		time.Sleep(time.Minute)
	}
}

func aria2StartDownload() {
	for {
		logs.GetLogger().Info("Start...")
		aria2Service.StartDownload(aria2Client, swanClient)
		logs.GetLogger().Info("Sleeping...")
		time.Sleep(time.Minute)
	}
}

func lotusStartImport() {
	for {
		logs.GetLogger().Info("Start...")
		lotusService.StartImport(swanClient)
		logs.GetLogger().Info("Sleeping...")
		time.Sleep(lotusService.ImportIntervalSecond)
	}
}

func lotusStartScan() {
	for {
		logs.GetLogger().Info("Start...")
		lotusService.StartScan(swanClient)
		logs.GetLogger().Info("Sleeping...")
		time.Sleep(lotusService.ScanIntervalSecond)
	}
}

func UpdateDealInfoAndLog(deal model.OfflineDeal, newSwanStatus string, filefullpath *string, messages ...string) {
	note := GetNote(messages...)
	if newSwanStatus == DEAL_STATUS_IMPORT_FAILED || newSwanStatus == DEAL_STATUS_DOWNLOAD_FAILED {
		logs.GetLogger().Warn(GetLog(deal, note))
	} else {
		logs.GetLogger().Info(GetLog(deal, note))
	}

	var updated bool
	var msg string
	if filefullpath != nil {
		if deal.Status == newSwanStatus && deal.Note == note && deal.FilePath == *filefullpath {
			logs.GetLogger().Info(GetLog(deal, constants.NOT_UPDATE_OFFLINE_DEAL_STATUS))
			return
		}

		msg = GetLog(deal, "set status to:"+newSwanStatus, "set note to:"+note, "set filepath to:"+*filefullpath)
		updated = swanClient.SwanUpdateOfflineDealStatus(deal.Id, newSwanStatus, note, *filefullpath)
	} else {
		if deal.Status == newSwanStatus && deal.Note == note {
			logs.GetLogger().Info(GetLog(deal, constants.NOT_UPDATE_OFFLINE_DEAL_STATUS))
			return
		}

		msg = GetLog(deal, "set status to:"+newSwanStatus, "set note to:"+note)
		updated = swanClient.SwanUpdateOfflineDealStatus(deal.Id, newSwanStatus, note)
	}

	if !updated {
		logs.GetLogger().Error(GetLog(deal, constants.UPDATE_OFFLINE_DEAL_STATUS_FAIL))
	} else {
		if newSwanStatus == DEAL_STATUS_IMPORT_FAILED || newSwanStatus == DEAL_STATUS_DOWNLOAD_FAILED {
			logs.GetLogger().Warn(msg)
		} else {
			logs.GetLogger().Info(msg)
		}
	}
}

func UpdateStatusAndLog(deal model.OfflineDeal, newSwanStatus string, messages ...string) {
	UpdateDealInfoAndLog(deal, newSwanStatus, nil, messages...)
}

func GetLog(deal model.OfflineDeal, messages ...string) string {
	text := GetNote(messages...)
	msg := fmt.Sprintf("deal(id=%d):%s,%s", deal.Id, deal.DealCid, text)
	return msg
}

func GetNote(messages ...string) string {
	result := ""
	if messages == nil {
		return result
	}
	for _, message := range messages {
		result = result + "," + message
	}

	result = strings.TrimPrefix(result, ",")
	result = strings.TrimSuffix(result, ",")
	return result
}
