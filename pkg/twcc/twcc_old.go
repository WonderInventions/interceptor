// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

// Package twcc provides interceptors to implement transport wide congestion control.
package twcc

import (
	"github.com/pion/rtcp"
)

type pktInfo struct {
	sequenceNumber uint32
	arrivalTime    int64
}

// recorderOld records incoming RTP packets and their delays and creates
// transport wide congestion control feedback reports as specified in
// https://datatracker.ietf.org/doc/html/draft-holmer-rmcat-transport-wide-cc-extensions-01
type recorderOld struct {
	receivedPackets []pktInfo

	cycles             uint32
	lastSequenceNumber uint16

	senderSSRC uint32
	mediaSSRC  uint32
	fbPktCnt   uint8
}

// newRecorderOld creates a new Recorder which uses the given senderSSRC in the created
// feedback packets.
func newRecorderOld(senderSSRC uint32) *recorderOld {
	return &recorderOld{
		receivedPackets: []pktInfo{},
		senderSSRC:      senderSSRC,
	}
}

// Record marks a packet with mediaSSRC and a transport wide sequence number sequenceNumber as received at arrivalTime.
func (r *recorderOld) Record(mediaSSRC uint32, sequenceNumber uint16, arrivalTime int64) {
	r.mediaSSRC = mediaSSRC
	if sequenceNumber < 0x0fff && (r.lastSequenceNumber&0xffff) > 0xf000 {
		r.cycles += 1 << 16
	}
	r.receivedPackets = insertSorted(r.receivedPackets, pktInfo{
		sequenceNumber: r.cycles | uint32(sequenceNumber),
		arrivalTime:    arrivalTime,
	})
	r.lastSequenceNumber = sequenceNumber
}

func insertSorted(list []pktInfo, element pktInfo) []pktInfo {
	if len(list) == 0 {
		return append(list, element)
	}
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].sequenceNumber < element.sequenceNumber {
			list = append(list, pktInfo{})
			copy(list[i+2:], list[i+1:])
			list[i+1] = element
			return list
		}
		if list[i].sequenceNumber == element.sequenceNumber {
			list[i] = element
			return list
		}
	}
	// element.sequenceNumber is between 0 and first ever received sequenceNumber
	return append([]pktInfo{element}, list...)
}

// BuildFeedbackPacket creates a new RTCP packet containing a TWCC feedback report.
func (r *recorderOld) BuildFeedbackPacket() []rtcp.Packet {
	if len(r.receivedPackets) < 2 {
		return nil
	}

	feedback := newFeedback(r.senderSSRC, r.mediaSSRC, r.fbPktCnt)
	r.fbPktCnt++
	feedback.setBase(uint16(r.receivedPackets[0].sequenceNumber&0xffff), r.receivedPackets[0].arrivalTime)

	var pkts []rtcp.Packet
	for _, pkt := range r.receivedPackets {
		ok := feedback.addReceived(uint16(pkt.sequenceNumber&0xffff), pkt.arrivalTime)
		if !ok {
			pkts = append(pkts, feedback.getRTCP())
			feedback = newFeedback(r.senderSSRC, r.mediaSSRC, r.fbPktCnt)
			r.fbPktCnt++
			feedback.addReceived(uint16(pkt.sequenceNumber&0xffff), pkt.arrivalTime)
		}
	}
	r.receivedPackets = []pktInfo{}
	pkts = append(pkts, feedback.getRTCP())

	return pkts
}
