package controllers

import (
	"context"
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"strings"
	"sync"
	"time"
)

const (
	failedStatus         = "FAILED"
	runningState         = "running"
	pausedState          = "paused"
	defaultWatchInterval = 30
	restartWaitPeriod    = 60
)

type ReplicationWatcher struct {
	Lock  *sync.Mutex
	state *string
}

func NewReplicationWatcher(lock *sync.Mutex) ReplicationWatcher {
	state := pausedState
	return ReplicationWatcher{
		Lock:  lock,
		state: &state,
	}
}

func (rw ReplicationWatcher) start(drr DisasterRecoveryReconciler, logger logr.Logger) {
	if *rw.state == runningState {
		return
	} else {
		*rw.state = runningState
	}
	logger.Info("Start Replication Watcher")
	interval := drr.cr.Spec.DisasterRecovery.ReplicationWatcherInterval
	if interval <= 0 {
		interval = defaultWatchInterval
	}
	go rw.watch(drr, logger, interval)
}

func (rw ReplicationWatcher) watch(drr DisasterRecoveryReconciler, logger logr.Logger, interval int) {
	for {
		if *rw.state == pausedState {
			logger.Info("Replication Watcher was stopped, exit from watch loop")
			return
		}
		// Fetch the OpenSearchService instance
		instance := &opensearchservice.OpenSearchService{}
		var err error
		if err = drr.reconciler.Client.Get(context.TODO(), types.NamespacedName{
			Namespace: drr.cr.Namespace,
			Name:      drr.cr.Name,
		}, instance); err != nil {
			logger.Error(err, "")
		} else {
			if instance.Spec.DisasterRecovery.Mode == "standby" &&
				instance.Status.DisasterRecoveryStatus.Mode == "standby" &&
				instance.Status.DisasterRecoveryStatus.Status == "done" {
				if err = rw.checkReplication(drr, logger); err != nil {
					logger.Info("Try to restart replication")
					rw.restartReplication(drr, logger)
				}
				if *rw.state == pausedState {
					logger.Info("Replication Watcher was stopped, exit from watch loop")
					return
				}
			}
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func (rw ReplicationWatcher) checkReplication(drr DisasterRecoveryReconciler, logger logr.Logger) error {
	defer rw.Lock.Unlock()
	rw.Lock.Lock()
	logger.Info("Start checking for replication status")
	replicationManager := drr.getReplicationManager()
	autoFollowRuleStats, err := replicationManager.GetAutoFollowRuleStats()
	if err != nil {
		logger.Error(err, "Cannot check autofollow replication rule")
	}
	if autoFollowRuleStats != nil {
		failedIndices := util.FilterSlice(autoFollowRuleStats.FailedIndices, func(s string) bool {
			return !strings.HasPrefix(s, ".")
		})
		if len(failedIndices) > 0 {
			logger.Info(fmt.Sprintf("Replication does not work correctly, there are failed_indices: %s", failedIndices))
			return fmt.Errorf("replication does not work correctly, there are failed_indices: %s", failedIndices)
		} else {
			indices, err := replicationManager.GetIndicesByPatternExcludeService(replicationManager.pattern)
			if err != nil {
				log.Error(err, fmt.Sprintf("Cannot get indices by pattern [%s]", replicationManager.pattern))
			} else {
				var failedReplications []string
				for _, index := range indices {
					if *rw.state == pausedState {
						logger.Info("Replication Watcher was stopped, exit from watch loop")
						return nil
					}
					replicationStatus, err := replicationManager.getIndexReplicationStatus(index)
					if err != nil {
						log.Error(err, fmt.Sprintf("Cannot get replication status of [%s] index", index))
					} else if replicationStatus.Status == failedStatus {
						failedReplications = append(failedReplications, index)
					} else if replicationStatus.Status == "PAUSED" {
						if strings.Contains(replicationStatus.Reason, "IndexNotFoundException") {
							logger.Info(fmt.Sprintf("Replication for index [%s] is paused because index was lost on active side, make sure active side has right content and remove standby index", index))
						} else {
							failedReplications = append(failedReplications, index)
						}
					}
				}
				if len(failedReplications) > 0 {
					logger.Info(fmt.Sprintf("Replication does not work correctly, there are failed_indices: %s", failedReplications))
					return fmt.Errorf("replication does not work correctly, there are failed_indices: %s", failedReplications)
				} else {
					logger.Info("Replication works correctly, there are no failed_indices")
				}
			}

		}
	} else {
		logger.Info("There is no autofollow rule")
		return fmt.Errorf("there is no autofollow rule")
	}
	return nil
}

func (rw ReplicationWatcher) pause(logger logr.Logger) {
	logger.Info("Stop Replication Watcher")
	*rw.state = pausedState
}

func (rw ReplicationWatcher) restartReplication(drr DisasterRecoveryReconciler, logger logr.Logger) {
	logger.Info("Restart replication")
	replicationManager := drr.getReplicationManager()
	err := drr.removePreviousReplication(replicationManager)
	if err != nil {
		logger.Error(err, "Previous replication cannot be stopped")
		return
	}
	err = drr.runReplicationProcess(replicationManager)
	if err != nil {
		logger.Error(err, "Replication cannot be started")
		return
	}
	logger.Info("Replication was restarted")
	time.Sleep(time.Second * restartWaitPeriod)
}
