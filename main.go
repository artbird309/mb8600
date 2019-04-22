package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/AdamJacobMuller/golib"
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

func main() {
	buf := bytes.NewBufferString(`{"GetMultipleHNAPs":{"GetMotoStatusDownstreamChannelInfo":"","GetMotoStatusUpstreamChannelInfo":""}}`)

	request, err := http.NewRequest("POST", "http://192.168.100.1/HNAP1/", buf)
	if err != nil {
		panic(err)
	}

	request.Header.Add("SOAPACTION", `"http://purenetworks.com/HNAP1/GetMultipleHNAPs"`)

	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	decodedResponse := &GetMultipleHNAPsResponseTop{}
	err = json.Unmarshal(body, decodedResponse)
	if err != nil {
		panic(err)
	}
	golib.JSON_PP(decodedResponse)

	downstreamChannels, err := ParseDownstreamChannelInfo(decodedResponse.GetMultipleHNAPsResponse.GetMotoStatusDownstreamChannelInfoResponse.MotoConnDownstreamChannel)
	if err != nil {
		panic(err)
	}
	golib.JSON_PP(downstreamChannels)

	upstreamChannels, err := ParseUpstreamChannelInfo(decodedResponse.GetMultipleHNAPsResponse.GetMotoStatusUpstreamChannelInfoResponse.MotoConnUpstreamChannel)
	if err != nil {
		panic(err)
	}
	golib.JSON_PP(upstreamChannels)
}
