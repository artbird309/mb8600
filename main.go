package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AdamJacobMuller/golib"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
)

var protocol = "https"

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

func NewInfluxBGWriter(influxClient client.Client, database string) (chan<- *client.Point, error) {
	pointChannel := make(chan *client.Point, 100000)

	go func(pointChannel <-chan *client.Point, influxClient client.Client, database string) {
		bp, err := client.NewBatchPoints(client.BatchPointsConfig{
			Database:  database,
			Precision: "s",
		})
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Panic("unable to create new influx batch points")
		}
		timer := time.Tick(time.Second)
		for {
			select {
			case v := <-pointChannel:
				bp.AddPoint(v)
			case <-timer:
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
				bp, err = client.NewBatchPoints(client.BatchPointsConfig{
					Database:  database,
					Precision: "s",
				})
				if err != nil {
					log.WithFields(log.Fields{
						"error":   err,
						"address": database,
					}).Error("unable to create new influx batch points")
					continue
				}
			}
		}
	}(pointChannel, influxClient, database)

	return pointChannel, nil
}

func main() {
	var influxdb_address, influxdb_database string
	flag.StringVar(&influxdb_address, "influxdb-address", "", "influxdb address")
	flag.StringVar(&influxdb_database, "influxdb-database", "", "influxdb database")
	flag.StringVar(&protocol, "protocol", "", "http or https for modem")
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
	bgChannel, err := NewInfluxBGWriter(influxClient, influxdb_database)

	for ts := range time.Tick(time.Second * 30) {
		start := time.Now()
		err, response := getModemData()
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("unable to make HTTP request")
			continue
		}

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("unable to read response body")
			continue
		}
		log.WithFields(log.Fields{
			"status": response.Status,
			"took":   time.Since(start),
		}).Info("got response")

		decodedResponse := &GetMultipleHNAPsResponseTop{}
		err = json.Unmarshal(body, decodedResponse)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("unable to unmarshal response body")
			continue
		}
		//golib.JSON_PP(decodedResponse)

		downstreamChannels, err := ParseDownstreamChannelInfo(decodedResponse.GetMultipleHNAPsResponse.GetMotoStatusDownstreamChannelInfoResponse.MotoConnDownstreamChannel)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("unable to parse downstream channel info")
			continue
		}
		//golib.JSON_PP(downstreamChannels)

		upstreamChannels, err := ParseUpstreamChannelInfo(decodedResponse.GetMultipleHNAPsResponse.GetMotoStatusUpstreamChannelInfoResponse.MotoConnUpstreamChannel)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("unable to parse upstream channel info")
			continue
		}
		//golib.JSON_PP(upstreamChannels)

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
				"frequency": fmt.Sprintf("%f", upstreamChannel.SignalCenter),
				"channelId": fmt.Sprintf("%d", upstreamChannel.Channel),
			}
			fields := map[string]interface{}{
				"symbolRate": upstreamChannel.SymbolRate,
				"power":      upstreamChannel.LaunchPower,
			}
			point, err := client.NewPoint("upstream", tags, fields, ts)
			if err != nil {
				log.WithFields(log.Fields{
					"error":  err,
					"tags":   tags,
					"fields": fields,
				}).Error("unable to create point for upstream_channels")
				continue
			}
			bgChannel <- point
		}

		tags := map[string]string{}
		fields := map[string]interface{}{
			"number": len(upstreamChannels),
		}
		point, err := client.NewPoint("upstreamBonded", tags, fields, ts)
		if err != nil {
			log.WithFields(log.Fields{
				"error":  err,
				"tags":   tags,
				"fields": fields,
			}).Error("unable to create point for upstreamBonded")
		}
		bgChannel <- point

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
				"frequency": fmt.Sprintf("%f", downstreamChannel.SignalCenter),
				"channelId": fmt.Sprintf("%d", downstreamChannel.Channel),
			}
			fields := map[string]interface{}{
				"power":       downstreamChannel.SignalStrength,
				"snr":         downstreamChannel.SNR,
				"corrected":   downstreamChannel.CorrectedErrors,
				"uncorrected": downstreamChannel.UncorrectedErrors,
			}
			point, err := client.NewPoint("downstream", tags, fields, ts)
			if err != nil {
				log.WithFields(log.Fields{
					"error":  err,
					"tags":   tags,
					"fields": fields,
				}).Error("unable to create point for downstream_channels")
				continue
			}
			bgChannel <- point
		}

		tags = map[string]string{}
		fields = map[string]interface{}{
			"number": len(downstreamChannels),
		}
		point, err = client.NewPoint("downstreamBonded", tags, fields, ts)
		if err != nil {
			log.WithFields(log.Fields{
				"error":  err,
				"tags":   tags,
				"fields": fields,
			}).Error("unable to create point for downstreamBonded")
		}
		bgChannel <- point
	}
}

func getModemData() (error, *http.Response) {
	buf := bytes.NewBufferString(`{"GetMultipleHNAPs":{"GetMotoStatusDownstreamChannelInfo":"","GetMotoStatusUpstreamChannelInfo":""}}`)
	request, err := http.NewRequest("POST", protocol + "://192.168.100.1/HNAP1/", buf)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err,
		}).Error("unable to create new HTTP request")
		return err, nil
	}
	request.Header.Add("SOAPACTION", `"http://purenetworks.com/HNAP1/GetMultipleHNAPs"`)
	switch protocol {
	case "https":
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient := &http.Client{Timeout: 2 * time.Second, Transport: tr}
		response, err := httpClient.Do(request)
		if err != nil {
			protocol = "http"
		}
		return err, response
	case "http":
		httpClient := http.Client{}
		response, err := httpClient.Do(request)
		return err, response
		if err != nil {
			protocol = "https"
		}
		return err, response
	default:
		return errors.New("unknown protocol"), nil
	}
}
