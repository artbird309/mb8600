package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AdamJacobMuller/golib"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
)

type GetMultipleHNAPsResponseTop struct {
	GetMultipleHNAPsResponse struct {
		GetMotoStatusDownstreamChannelInfoResponse struct {
			MotoConnDownstreamChannel                string `json:"MotoConnDownstreamChannel"`
			GetMotoStatusDownstreamChannelInfoResult string `json:"GetMotoStatusDownstreamChannelInfoResult"`
		} `json:"GetMotoStatusDownstreamChannelInfoResponse"`
		GetMotoStatusUpstreamChannelInfoResponse struct {
			MotoConnUpstreamChannel                string `json:"MotoConnUpstreamChannel"`
			GetMotoStatusUpstreamChannelInfoResult string `json:"GetMotoStatusUpstreamChannelInfoResult"`
		} `json:"GetMotoStatusUpstreamChannelInfoResponse"`
		GetMultipleHNAPsResult string `json:"GetMultipleHNAPsResult"`
	} `json:"GetMultipleHNAPsResponse"`
}

// 1^Locked^QAM256^3^477.0^ 4.4^40.9^2135^0^
type DownstreamChannelInfo struct {
	Channel    int    // 0
	Status     string // 1
	Modulation string // 2

	CMTSChannel       int     // 3
	SignalCenter      float64 // 4
	SignalStrength    float64 // 5
	SNR               float64 // 6
	CorrectedErrors   int     // 7
	UncorrectedErrors int     // 8
}

func ParseDownstreamChannelInfo(ci string) ([]*DownstreamChannelInfo, error) {
	var channels []*DownstreamChannelInfo

	channelStrings := strings.Split(ci, "|+|")

	for _, channelString := range channelStrings {
		if channelString == "" {
			continue
		}
		channel := &DownstreamChannelInfo{}
		channelStringSplit := strings.Split(channelString, "^")
		if len(channelStringSplit) != 10 {
			return nil, errors.New("channelString does not have 10 parts: " + channelString)
		}

		channelNumber, err := strconv.ParseInt(channelStringSplit[0], 10, 64)
		if err != nil {
			return nil, err
		}
		channel.Channel = int(channelNumber)

		channel.Status = channelStringSplit[1]
		channel.Modulation = channelStringSplit[2]

		CMTSChannel, err := strconv.ParseInt(channelStringSplit[3], 10, 64)
		if err != nil {
			return nil, err
		}
		channel.CMTSChannel = int(CMTSChannel)

		SignalCenter, err := strconv.ParseFloat(channelStringSplit[4], 64)
		if err != nil {
			return nil, err
		}
		channel.SignalCenter = SignalCenter

		SignalStrength, err := strconv.ParseFloat(strings.Trim(channelStringSplit[5], " "), 64)
		if err != nil {
			return nil, err
		}
		channel.SignalStrength = SignalStrength

		SNR, err := strconv.ParseFloat(channelStringSplit[6], 64)
		if err != nil {
			return nil, err
		}
		channel.SNR = SNR

		CorrectedErrors, err := strconv.ParseInt(channelStringSplit[7], 10, 64)
		if err != nil {
			return nil, err
		}
		channel.CorrectedErrors = int(CorrectedErrors)

		UncorrectedErrors, err := strconv.ParseInt(channelStringSplit[8], 10, 64)
		if err != nil {
			return nil, err
		}
		channel.UncorrectedErrors = int(UncorrectedErrors)

		channels = append(channels, channel)
	}

	return channels, nil
}

// 1^Locked^SC-QAM^1^5120^35.8^35.0^
type UpstreamChannelInfo struct {
	Channel    int    // 0
	Status     string // 1
	Modulation string // 2

	CMTSChannel  int     // 3
	SymbolRate   int     // 4
	SignalCenter float64 // 5
	LaunchPower  float64 // 6
}

func ParseUpstreamChannelInfo(ci string) ([]*UpstreamChannelInfo, error) {
	var channels []*UpstreamChannelInfo

	channelStrings := strings.Split(ci, "|+|")

	for _, channelString := range channelStrings {
		if channelString == "" {
			continue
		}
		channel := &UpstreamChannelInfo{}
		channelStringSplit := strings.Split(channelString, "^")
		if len(channelStringSplit) != 8 {
			return nil, errors.New("channelString does not have 10 parts: " + channelString)
		}

		channelNumber, err := strconv.ParseInt(channelStringSplit[0], 10, 64)
		if err != nil {
			return nil, err
		}
		channel.Channel = int(channelNumber)

		channel.Status = channelStringSplit[1]
		channel.Modulation = channelStringSplit[2]

		CMTSChannel, err := strconv.ParseInt(channelStringSplit[3], 10, 64)
		if err != nil {
			return nil, err
		}
		channel.CMTSChannel = int(CMTSChannel)

		SymbolRate, err := strconv.ParseInt(channelStringSplit[4], 10, 64)
		if err != nil {
			return nil, err
		}
		channel.SymbolRate = int(SymbolRate)

		SignalCenter, err := strconv.ParseFloat(strings.Trim(channelStringSplit[5], " "), 64)
		if err != nil {
			return nil, err
		}
		channel.SignalCenter = SignalCenter

		LaunchPower, err := strconv.ParseFloat(channelStringSplit[6], 64)
		if err != nil {
			return nil, err
		}
		channel.LaunchPower = LaunchPower

		channels = append(channels, channel)
	}

	return channels, nil
}

func x() {
	_ = golib.JSON_PP
}

func main() {
	httpClient := &http.Client{}

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	var mac, influxdb_address, influxdb_database string
	flag.StringVar(&mac, "mac", "", "mac address of router")
	flag.StringVar(&influxdb_address, "influxdb-address", "", "influxdb address")
	flag.StringVar(&influxdb_database, "influxdb-database", "", "influxdb database")
	flag.Parse()

	influxClient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxdb_address,
	})
	if err != nil {
		log.WithFields(log.Fields{
			"error":   err,
			"address": influxdb_address,
		}).Panic("unable to create new influx HTTP client")
	}

	for ts := range time.Tick(time.Second * 30) {
		start := time.Now()
		buf := bytes.NewBufferString(`{"GetMultipleHNAPs":{"GetMotoStatusDownstreamChannelInfo":"","GetMotoStatusUpstreamChannelInfo":""}}`)

		request, err := http.NewRequest("POST", "http://192.168.100.1/HNAP1/", buf)
		if err != nil {
			panic(err)
		}

		request.Header.Add("SOAPACTION", `"http://purenetworks.com/HNAP1/GetMultipleHNAPs"`)

		response, err := httpClient.Do(request)
		if err != nil {
			panic(err)
		}

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			panic(err)
		}
		log.WithFields(log.Fields{
			"status": response.Status,
			"took":   time.Since(start),
		}).Info("got response")

		decodedResponse := &GetMultipleHNAPsResponseTop{}
		err = json.Unmarshal(body, decodedResponse)
		if err != nil {
			panic(err)
		}
		//golib.JSON_PP(decodedResponse)

		downstreamChannels, err := ParseDownstreamChannelInfo(decodedResponse.GetMultipleHNAPsResponse.GetMotoStatusDownstreamChannelInfoResponse.MotoConnDownstreamChannel)
		if err != nil {
			panic(err)
		}
		//golib.JSON_PP(downstreamChannels)

		upstreamChannels, err := ParseUpstreamChannelInfo(decodedResponse.GetMultipleHNAPsResponse.GetMotoStatusUpstreamChannelInfoResponse.MotoConnUpstreamChannel)
		if err != nil {
			panic(err)
		}
		//golib.JSON_PP(upstreamChannels)

		bp, err := client.NewBatchPoints(client.BatchPointsConfig{
			Database:  influxdb_database,
			Precision: "s",
		})
		if err != nil {
			log.WithFields(log.Fields{
				"error":    err,
				"address":  influxdb_address,
				"database": influxdb_database,
			}).Panic("unable to create new influx batch points")
		}

		/*
			type UpstreamChannelInfo struct {
				Channel    int    // 0
				Status     string // 1
				Modulation string // 2

				CMTSChannel  int     // 3
				SymbolRate   int     // 4
				SignalCenter float64 // 5
				LaunchPower  float64 // 6
			}

		*/
		for _, upstreamChannel := range upstreamChannels {
			tags := map[string]string{
				"hostname":      hostname,
				"mac":           mac,
				"signal-center": fmt.Sprintf("%f", upstreamChannel.SignalCenter),
				"channel":       fmt.Sprintf("%d", upstreamChannel.Channel),
			}
			fields := map[string]interface{}{
				"cmts-channel":  upstreamChannel.CMTSChannel,
				"signal-center": upstreamChannel.SignalCenter,
				"symbol-rate":   upstreamChannel.SymbolRate,
				"launch-power":  upstreamChannel.LaunchPower,
			}
			point, err := client.NewPoint("upstream_channels", tags, fields, ts)
			if err != nil {
				panic(err)
			}
			bp.AddPoint(point)
		}

		/*
			// 1^Locked^QAM256^3^477.0^ 4.4^40.9^2135^0^
			type DownstreamChannelInfo struct {
				Channel    int    // 0
				Status     string // 1
				Modulation string // 2

				CMTSChannel       int     // 3
				SignalCenter      float64 // 4
				SignalStrength    float64 // 5
				SNR               float64 // 6
				CorrectedErrors   int     // 7
				UncorrectedErrors int     // 8
			}
		*/
		for _, downstreamChannel := range downstreamChannels {
			tags := map[string]string{
				"hostname":      hostname,
				"mac":           mac,
				"signal-center": fmt.Sprintf("%f", downstreamChannel.SignalCenter),
				"channel":       fmt.Sprintf("%d", downstreamChannel.Channel),
			}
			fields := map[string]interface{}{
				"cmts-channel":       downstreamChannel.CMTSChannel,
				"signal-center":      downstreamChannel.SignalCenter,
				"signal-strength":    downstreamChannel.SignalStrength,
				"snr":                downstreamChannel.SNR,
				"corrected-errors":   downstreamChannel.CorrectedErrors,
				"uncorrected-errors": downstreamChannel.UncorrectedErrors,
			}
			point, err := client.NewPoint("downstream_channels", tags, fields, ts)
			if err != nil {
				panic(err)
			}
			bp.AddPoint(point)
		}

		bpc := len(bp.Points())
		if bpc == 0 {
			continue
		} else {
			log.WithFields(log.Fields{
				"points": bpc,
			}).Info("writing to influxdb")
		}

		err = influxClient.Write(bp)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("writing to influxdb failed")
			continue
		}

	}
}
