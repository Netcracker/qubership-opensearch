This section deals with troubleshooting of data files if they are corrupted on both primary and replica shards.

Sometimes some shards are corrupted in pairs such as primary shard and corresponding replica shard, so you can not use replica shards approach. For more information, refer to Elasticsearch reference: [https://www.elastic.co/guide/en/elasticsearch/guide/master/replica-shards.html](https://www.elastic.co/guide/en/elasticsearch/guide/master/replica-shards.html).

In this case you should use backups for recovering data, but if there are no backups and data loss is acceptable you can relocate shards.

**Note**: This procedure leads to data loss.

# OpenSearch Metric

This section describes how to retrieve information about the shards.

## Check the Indices

To retrieve the information about all indices in the cluster run the following command:

```
curl -XGET http://localhost:9200/_cat/indices?v
```

Possible output:

```
health status index                uuid                   pri rep docs.count docs.deleted store.size pri.store.size
green  open   dbaas_metadata       RQluYUSAQmaMid5RFMU3GA   1   1          1            0        6kb            3kb
green  open   .kibana_1            rCvMZI4ET8ichepIyUWNPg   1   1          0            0       416b           208b
green  open   .opendistro_security DQZhHXZeRpeIwnEEzjMadg   1   2          9            0    126.5kb         42.1kb
```

If you have indices in red status, you should check the corresponding shards.

## Check the Shards

To retrieve information about all shards in the cluster, run the following command:

```
curl -X GET http://localhost:9200/_cat/shards?v
```

Possible output:

```
index                shard prirep state   docs  store ip           node
.kibana_1            0     r      STARTED    0   208b 10.128.7.6   opensearch-1
.kibana_1            0     p      STARTED    0   208b 10.129.6.154 opensearch-0
.opendistro_security 0     p      STARTED    9 42.1kb 10.128.7.6   opensearch-1
.opendistro_security 0     r      STARTED    9 42.1kb 10.129.6.154 opensearch-0
.opendistro_security 0     r      STARTED    9 42.1kb 10.130.5.131 opensearch-2
dbaas_metadata       0     p      STARTED    1    3kb 10.128.7.6   opensearch-1
dbaas_metadata       0     r      STARTED    1    3kb 10.130.5.131 opensearch-2
```
    

If data files are corrupted on both primary and replica shards, in this example shards number 2 and 4, you should use the backups for recovering data.

If there are no backups and data loss isn't sensitive you can relocate unassigned shards to available node, for example to data-third, maybe some data is lost but the index has green status.

# Troubleshooting Procedure

**Note**: Some indices data will be lost. If you have backups use them.

Make sure that some indices have `red` status. Make sure all of unassigned shards are corrupted in pairs, primary and replica shard.

## Prerequisites

* bash is available.
* curl [https://curl.haxx.se](https://curl.haxx.se) is available.

Save the following as relocate_shards.sh script:

```
        #!/usr/bin/env bash
        
        
        # URL of Elastic Search
        url=
        # To this node shards will be relocated
        node_name=
        username=
        password=
        
        post_query(){
        	index=$1
        	shard=$2
        	node=$3
        	curl -u "$username:$password" -XPOST ${url}"/_cluster/reroute" -d "{  \"commands\" : [{ \"allocate_empty_primary\" : {\"index\" : \"${index}\", \"shard\" : ${shard}, \"node\" : \"${node}\"}}] }"
        }
        
        relocate(){
          url1=${url1}
          node=${node_name}
          while read data; do
            pairs=$(echo $data | tr " " "\n")
        	array=()
        	for pair in ${pairs}; do
        		array+=(${pair})
        	done
        	index=${array[0]}
        	shard=${array[1]}
        	post_query ${index} ${shard} ${node} 2>& 1 >/dev/null
          done
        }
        
        curl -u "$username:$password" $url/_cat/shards | grep UNASSIGNED \
        		  | awk -v node=${node_name} -v url=${url} '{if ( $3 == "p" ) \
									                {print $1, $2}}' | relocate         
```
Make sure the script file can be executed, do `chmod +x` on it:

        chmod +x relocate_shards.sh

Specify the OpenSearch server URL in the `url` parameter in the `relocate_shards.sh` script.

Specify the name of OpenSearch available node in the `node_name` parameter in the `relocate_shards.sh` script.

Specify the OpenSearch `username` and `password` parameter in the `relocate_shards.sh` script.

Make sure you have OpenSearch backups, or data loss is acceptable.

Run the `relocate_shards.sh` script.

Perform request to the URL `OS_URL/_cat/shards` after the script `relocate_shards.sh` is executed.

You can use your browser or use the following curl command:

```
curl OS_URL/_cat/shards
```

Expected output:

```
        index                         shard   status      docs  memory    ip       node

        cats                              4 r STARTED      0     159b 10.1.6.185 data-third
        cats                              4 p STARTED      0     159b 10.1.10.27 data-second
        cats                              2 r STARTED      2   41.3kb 10.1.5.47  data-first
        cats                              2 p STARTED      2   41.3kb 10.1.10.27 data-second
        cats                              1 r STARTED      1   37.7kb 10.1.6.185 data-third
        cats                              1 p STARTED      1   37.7kb 10.1.5.47  data-first
        cats                              3 r STARTED      1   37.7kb 10.1.6.185 data-third
        cats                              3 p STARTED      1   37.7kb 10.1.10.27 data-second
        cats                              0 r STARTED      0     159b 10.1.5.47  data-first
        cats                              0 p STARTED      0     159b 10.1.10.27 data-second
```
You need to make sure there is no "UNASSIGNED" status for any shard.
