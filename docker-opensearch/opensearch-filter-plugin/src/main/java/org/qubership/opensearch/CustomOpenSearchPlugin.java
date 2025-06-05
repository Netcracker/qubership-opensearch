/*
 * Copyright 2024-2025 NetCracker Technology Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package org.qubership.opensearch;

import java.util.ArrayList;
import java.util.Collection;
import java.util.List;
import java.util.Objects;
import java.util.function.Supplier;
import org.opensearch.action.support.ActionFilter;
import org.opensearch.client.Client;
import org.opensearch.cluster.metadata.IndexNameExpressionResolver;
import org.opensearch.cluster.service.ClusterService;
import org.opensearch.core.common.io.stream.NamedWriteableRegistry;
import org.opensearch.core.xcontent.NamedXContentRegistry;
import org.opensearch.env.Environment;
import org.opensearch.env.NodeEnvironment;
import org.opensearch.plugins.ActionPlugin;
import org.opensearch.plugins.Plugin;
import org.opensearch.repositories.RepositoriesService;
import org.opensearch.script.ScriptService;
import org.opensearch.threadpool.ThreadPool;
import org.opensearch.watcher.ResourceWatcherService;

public class CustomOpenSearchPlugin extends Plugin implements ActionPlugin {

  private volatile Client client;
  private volatile ThreadPool threadPool;

  @Override
  public List<ActionFilter> getActionFilters() {
    List<ActionFilter> filters = new ArrayList<>(1);
    ActionFilter securityFilter = new IsmSecurityFilter(threadPool, client);
    filters.add(Objects.requireNonNull(securityFilter));
    return filters;
  }

  @Override
  public Collection<Object> createComponents(Client client, ClusterService clusterService,
      ThreadPool threadPool, ResourceWatcherService resourceWatcherService,
      ScriptService scriptService, NamedXContentRegistry namedXContentRegistry,
      Environment environment, NodeEnvironment nodeEnvironment,
      NamedWriteableRegistry namedWriteableRegistry,
      IndexNameExpressionResolver indexNameExpressionResolver,
      Supplier<RepositoriesService> repositoriesServiceSupplier) {
    this.threadPool = threadPool;
    this.client = client;
    return super.createComponents(client, clusterService, threadPool, resourceWatcherService,
        scriptService, namedXContentRegistry, environment, nodeEnvironment, namedWriteableRegistry,
        indexNameExpressionResolver, repositoriesServiceSupplier);
  }
}
