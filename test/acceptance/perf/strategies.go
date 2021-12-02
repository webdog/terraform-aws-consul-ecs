package perf

import (
	"fmt"
	"time"
)

type stabilizationEvent struct {
	time     time.Time
	duration time.Duration
}

type Strategy interface {
	ServiceGroupsToDelete([]int, time.Time) []int
	setServiceGroupStable(int, time.Time)
	Data() []stabilizationEvent
	Done() bool
}

type EverythingStabilizes struct {
	restarts                 int
	serviceGroups            []int
	initialStabilizationTime time.Time
	subsequentStabilizations []time.Time
}

func NewEverythingStabilizes(serviceGroups int, restarts int) Strategy {
	var allServiceGroups []int
	for i := 0; i < serviceGroups; i++ {
		allServiceGroups = append(allServiceGroups, i)
	}
	return &EverythingStabilizes{
		serviceGroups: allServiceGroups,
		restarts:      restarts,
	}
}

func (sgs *EverythingStabilizes) ServiceGroupsToDelete(stableServiceGroups []int, t time.Time) []int {
	var emptyServiceGroups []int

	if len(sgs.serviceGroups) != len(stableServiceGroups) || len(sgs.subsequentStabilizations) == sgs.restarts {
		return emptyServiceGroups
	}

	sgs.setServiceGroupStable(0, t)

	fmt.Println("Every service group is healthy")

	return stableServiceGroups
}

func (sgs *EverythingStabilizes) Done() bool {
	return len(sgs.subsequentStabilizations) == sgs.restarts
}

func (es *EverythingStabilizes) Data() []stabilizationEvent {
	var timeseries []stabilizationEvent

	for i, time := range es.subsequentStabilizations {
		if i == 0 {
			timeseries = append(timeseries, stabilizationEvent{time, time.Sub(es.initialStabilizationTime)})
		} else {
			timeseries = append(timeseries, stabilizationEvent{time, time.Sub(es.subsequentStabilizations[i-1])})
		}
	}

	return timeseries
}

func (es *EverythingStabilizes) setServiceGroupStable(serviceGroup int, t time.Time) {
	if es.initialStabilizationTime.IsZero() {
		es.initialStabilizationTime = t
	} else {
		var duration time.Duration
		if len(es.subsequentStabilizations) == 0 {
			duration = t.Sub(es.initialStabilizationTime)
		} else {
			duration = t.Sub(es.subsequentStabilizations[len(es.subsequentStabilizations)-1])
		}
		RecordDuration(duration, serviceGroup)

		es.subsequentStabilizations = append(es.subsequentStabilizations, t)
	}
}

type ServiceGroupStabilizes struct {
	restarts                  int
	serviceGroups             []int
	initialStabilizationTimes map[int]time.Time
	subsequentStabilizations  map[int][]time.Time
}

func NewServiceGroupStabilizes(serviceGroups int, restarts int) Strategy {
	var allServiceGroups []int
	initialStabilizationTimes := make(map[int]time.Time)
	subsequentStabilizations := make(map[int][]time.Time)

	for i := 0; i < serviceGroups; i++ {
		allServiceGroups = append(allServiceGroups, i)
		initialStabilizationTimes[i] = time.Time{}
		subsequentStabilizations[i] = []time.Time{}
	}

	return &ServiceGroupStabilizes{
		restarts:                  restarts,
		serviceGroups:             allServiceGroups,
		initialStabilizationTimes: initialStabilizationTimes,
		subsequentStabilizations:  subsequentStabilizations,
	}
}

func (sgs *ServiceGroupStabilizes) ServiceGroupsToDelete(stableServiceGroups []int, t time.Time) []int {
	var toRestart []int

	for _, serviceGroup := range stableServiceGroups {
		if len(sgs.subsequentStabilizations[serviceGroup]) != sgs.restarts {
			sgs.setServiceGroupStable(serviceGroup, time.Now())
			toRestart = append(toRestart, serviceGroup)
		}
	}

	return toRestart
}

func (sgs *ServiceGroupStabilizes) Data() []stabilizationEvent {
	var timeseries []stabilizationEvent
	for serviceGroupIndex, times := range sgs.subsequentStabilizations {
		for i := 0; i < len(times); i++ {
			if i == 0 {
				timeseries = append(timeseries, stabilizationEvent{times[i], times[i].Sub(sgs.initialStabilizationTimes[serviceGroupIndex])})
			} else {
				timeseries = append(timeseries, stabilizationEvent{times[i], times[i].Sub(times[i-1])})
			}
		}
	}
	return timeseries
}

func (sgs *ServiceGroupStabilizes) Done() bool {
	for _, times := range sgs.subsequentStabilizations {
		if len(times) < sgs.restarts {
			return false
		}
	}
	return true
}

func (es *ServiceGroupStabilizes) setServiceGroupStable(serviceGroup int, t time.Time) {
	if es.initialStabilizationTimes[serviceGroup].IsZero() {
		es.initialStabilizationTimes[serviceGroup] = t
	} else {
		var duration time.Duration
		if len(es.subsequentStabilizations[serviceGroup]) == 0 {
			duration = t.Sub(es.initialStabilizationTimes[serviceGroup])
		} else {
			duration = t.Sub(es.subsequentStabilizations[serviceGroup][len(es.subsequentStabilizations[serviceGroup])-1])
		}
		RecordDuration(duration, serviceGroup)

		es.subsequentStabilizations[serviceGroup] = append(es.subsequentStabilizations[serviceGroup], t)
	}
}
