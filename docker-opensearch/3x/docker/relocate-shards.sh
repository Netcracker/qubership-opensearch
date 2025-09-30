#!/usr/bin/env bash


relocate(){
  len=${#nodes[@]}
  while read data; do
	IFS=" " read -r -a pair <<< ${data}
	index=${pair[0]}
	shard=${pair[1]}
	node=${nodes[$num]}
	num=$(( (${num} + 1) % ${len}))
	curl -k -u ${OPENSEARCH_USERNAME}:${OPENSEARCH_PASSWORD} -XPOST ${url}"/_cluster/reroute" -H 'Content-Type: application/json' -d "{  \"commands\" : [{ \"allocate_empty_primary\" : {\"index\" : \"${index}\", \"shard\" : ${shard}, \"node\" : \"${node}\", \"accept_data_loss\" : true }}] }"
  done
}

url="localhost:9200"
num=0
nodes=()
index=$1

old_ifs=$IFS
while IFS='' read -r node; do nodes+=("$node"); done < <(curl -k -u ${OPENSEARCH_USERNAME}:${OPENSEARCH_PASSWORD} ${url}/_cat/nodes?h=name)

curl -k -u ${OPENSEARCH_USERNAME}:${OPENSEARCH_PASSWORD} ${url}"/_cat/shards/"${index} | grep UNASSIGNED \
		  | awk '{if ( $3 == "p" ) {print $1, $2}}' | relocate

IFS=${old_ifs}