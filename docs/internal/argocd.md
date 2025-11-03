This chapter describes the argoCD configuration procedures of OpenSearch.

# Sync

Once you fetch your application in argocd then click on Sync button and then Synchronize, your application sync will start.

* If you face Sync Failed issue like : 

```text
one or more objects failed to apply, GrafanaDashboard.integreatly.org "opensearch-grafana-dashboard" is invalid: metadata.annotations: Too long: must have at most 262144 bytes
Then you can see 'Server-Side Apply' while click on sync button, check it and sync it again.
```

# Out Of Sync

If you face issue with out of sync, check the difference.

* If you see the difference in duration like :
    
    ```yaml
    duration: 365h
    ```

    Diff : 
    
    ```yaml
    duration: 365h0m0s
    ```

    Then you need to add 0m0s in duration of yaml.

* If you see the difference in isCA like :
    
    ```yaml
    isCA: false
    ```
    
    Then please remove it because it is not required if it is false.

* If you see difference of monitoring resources 'kind: AmsObserverGroup' like below.
    
    ```yaml 
    apiVersion: monitoring.netcracker.com/v1beta1
    kind: AmsObserverGroup
    metadata:
        kubectl.kubernetes.io/last-applied-configuration:

    ```
    
    Then you can add annotations to ingore trackking this resources and then sync it:
    
    ```yaml
    apiVersion: monitoring.netcracker.com/v1beta1
    kind: AmsObserverGroup
    metadata:
        annotations:
            argocd.argoproj.io/compare-options: IgnoreExtraneous
        kubectl.kubernetes.io/last-applied-configuration:
    ```
