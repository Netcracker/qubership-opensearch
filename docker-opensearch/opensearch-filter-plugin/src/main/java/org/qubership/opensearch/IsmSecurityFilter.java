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

import static org.qubership.opensearch.Constants.ADD_POLICY_ACTION_NAME;
import static org.qubership.opensearch.Constants.CHANGE_POLICY_ACTION_NAME;
import static org.qubership.opensearch.Constants.DELETE_POLICY_ACTION_NAME;
import static org.qubership.opensearch.Constants.EXPLAIN_ACTION_NAME;
import static org.qubership.opensearch.Constants.GET_POLICY_ACTION_NAME;
import static org.qubership.opensearch.Constants.INDEX_POLICY_ACTION_NAME;
import static org.qubership.opensearch.Constants.ISM_ACTION_PATTERN;
import static org.qubership.opensearch.Constants.ISM_CONFIG_INDEX_NAME;
import static org.qubership.opensearch.Constants.ISM_ROLE_NAME;
import static org.qubership.opensearch.Constants.OPENDISTRO_SECURITY_USER;
import static org.qubership.opensearch.Constants.REMOVE_POLICY_ACTION_NAME;
import static org.qubership.opensearch.Constants.RESOURCE_PREFIX_ATTRIBUTE_NAME;
import static org.qubership.opensearch.Constants.RETRY_FAILED_MANAGED_INDEX_ACTION_NAME;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.google.common.annotations.VisibleForTesting;
import java.io.IOException;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.util.Arrays;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ExecutionException;
import java.util.stream.StreamSupport;
import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.opensearch.OpenSearchSecurityException;
import org.opensearch.action.ActionRequest;
import org.opensearch.action.search.SearchRequest;
import org.opensearch.action.search.SearchResponse;
import org.opensearch.action.support.ActionFilter;
import org.opensearch.action.support.ActionFilterChain;
import org.opensearch.client.Client;
import org.opensearch.common.action.ActionFuture;
import org.opensearch.common.io.stream.BytesStreamOutput;
import org.opensearch.common.util.concurrent.ThreadContext;
import org.opensearch.common.util.concurrent.ThreadContext.StoredContext;
import org.opensearch.common.xcontent.XContentHelper;
import org.opensearch.common.xcontent.XContentType;
import org.opensearch.core.action.ActionListener;
import org.opensearch.core.action.ActionResponse;
import org.opensearch.core.common.bytes.BytesReference;
import org.opensearch.core.common.io.stream.StreamInput;
import org.opensearch.core.common.io.stream.Writeable;
import org.opensearch.core.rest.RestStatus;
import org.opensearch.core.xcontent.MediaType;
import org.opensearch.core.xcontent.ToXContent;
import org.opensearch.core.xcontent.ToXContentObject;
import org.opensearch.index.query.BoolQueryBuilder;
import org.opensearch.index.query.IdsQueryBuilder;
import org.opensearch.index.query.RegexpQueryBuilder;
import org.opensearch.index.query.TermQueryBuilder;
import org.opensearch.index.reindex.BulkByScrollResponse;
import org.opensearch.index.reindex.UpdateByQueryAction;
import org.opensearch.index.reindex.UpdateByQueryRequest;
import org.opensearch.script.Script;
import org.opensearch.script.ScriptType;
import org.opensearch.search.SearchHit;
import org.opensearch.tasks.Task;
import org.opensearch.threadpool.ThreadPool;
import org.qubership.opensearch.security.user.User;

public class IsmSecurityFilter implements ActionFilter {

  private final Logger log = LogManager.getLogger(this.getClass());

  private static final String SECURITY_ERROR_PATTERN = "no permissions for [%s] and %s";

  private final Client client;
  private final ThreadContext threadContext;

  public IsmSecurityFilter(ThreadPool threadPool, Client client) {
    this.threadContext = threadPool.getThreadContext();
    this.client = client;
  }

  @Override
  public int order() {
    return Integer.MIN_VALUE;
  }

  @Override
  public <RequestT extends ActionRequest, ResponseT extends ActionResponse> void apply(Task task,
      String action, RequestT request, ActionListener<ResponseT> listener,
      ActionFilterChain<RequestT, ResponseT> actionFilterChain) {
    if (action == null || !action.startsWith(ISM_ACTION_PATTERN)) {
      actionFilterChain.proceed(task, action, request, listener);
      return;
    }
    try (StoredContext ignored = threadContext.newStoredContext(true)) {
      applyRequest(task, action, request, listener, actionFilterChain);
    }
  }

  /**
   * Filters the execution of an action on the request side.
   */
  private <RequestT extends ActionRequest, ResponseT extends ActionResponse> void applyRequest(
      Task task, String action, RequestT request, ActionListener<ResponseT> actionListener,
      ActionFilterChain<RequestT, ResponseT> actionFilterChain) {
    // Get user information from thread context
    User user = null;
    try {
      Writeable contextUser = threadContext.getTransient(OPENDISTRO_SECURITY_USER);
      if (contextUser != null) {
        user = new User(getStreamInput(contextUser));
      }
    } catch (IOException e) {
      throw new RuntimeException(e);
    }
    if (user == null) {
      actionFilterChain.proceed(task, action, request, actionListener);
      return;
    }

    String resourcePrefixAttribute = user.getAttributes().get(RESOURCE_PREFIX_ATTRIBUTE_NAME);
    if (user.getBackendRoles().contains(ISM_ROLE_NAME) && resourcePrefixAttribute != null) {
      try {
        if (allowAction(action, request, resourcePrefixAttribute)) {
          ActionListener<ResponseT> overriddenActionListener =
              overrideActionListener(action, request, actionListener);
          actionFilterChain.proceed(task, action, request, overriddenActionListener);
        } else {
          String errorMessage = String.format(SECURITY_ERROR_PATTERN, action, user);
          log.error(errorMessage);
          actionListener.onFailure(
              new OpenSearchSecurityException(errorMessage, RestStatus.FORBIDDEN));
        }
        return;
      } catch (IOException e) {
        throw new RuntimeException(e);
      }
    }
    actionFilterChain.proceed(task, action, request, actionListener);
  }

  /**
   * Whether specified action is permitted.
   *
   * @param action         the name of action
   * @param request        the request information
   * @param resourcePrefix the resource prefix
   * @return true if operation is permitted, false otherwise
   * @throws IOException if there is an issue with reading from StreamInput
   */
  @VisibleForTesting
  protected boolean allowAction(String action, ActionRequest request, String resourcePrefix)
      throws IOException {
    StreamInput in = getStreamInput(request);
    switch (action) {
      case ADD_POLICY_ACTION_NAME:
        return allowAddPolicyAction(in, resourcePrefix);
      case CHANGE_POLICY_ACTION_NAME:
        return allowChangePolicyAction(in, resourcePrefix);
      case DELETE_POLICY_ACTION_NAME:
        return allowDeletePolicyAction(in, resourcePrefix);
      case EXPLAIN_ACTION_NAME:
        return allowExplainAction(in, resourcePrefix);
      case GET_POLICY_ACTION_NAME:
        return allowGetPolicyAction(in, resourcePrefix);
      case INDEX_POLICY_ACTION_NAME:
        return allowIndexPolicyAction(request, resourcePrefix);
      case REMOVE_POLICY_ACTION_NAME:
        return allowRemovePolicyAction(in, resourcePrefix);
      case RETRY_FAILED_MANAGED_INDEX_ACTION_NAME:
        return allowRetryFailedManagedIndexAction(in, resourcePrefix);
      default:
        return false;
    }
  }

  /**
   * Returns the original or overridden action listener.
   *
   * @param action         the name of action
   * @param request        the request information
   * @param actionListener the action listener
   * @return calculated action listener
   * @throws IOException if there is an issue with reading from StreamInput
   */
  private <ResponseT extends ActionResponse> ActionListener<ResponseT> overrideActionListener(
      String action, ActionRequest request, ActionListener<ResponseT> actionListener)
      throws IOException {
    if (!INDEX_POLICY_ACTION_NAME.equals(action)
        && !ADD_POLICY_ACTION_NAME.equals(action)
        && !CHANGE_POLICY_ACTION_NAME.equals(action)) {
      return actionListener;
    }
    StreamInput in = getStreamInput(request);
    BoolQueryBuilder combinedQuery = new BoolQueryBuilder();
    String scriptCode = "";
    if (INDEX_POLICY_ACTION_NAME.equals(action)) {
      String policyId = in.readString();
      combinedQuery.must(new IdsQueryBuilder().addIds(policyId));
      scriptCode = "ctx._source.policy.user = params.user";
    }
    if (ADD_POLICY_ACTION_NAME.equals(action)) {
      List<String> indices = in.readStringList();
      String policyId = in.readString();
      combinedQuery.must(new TermQueryBuilder("managed_index.policy_id", policyId));
      combinedQuery.must(new RegexpQueryBuilder("managed_index.index",
          String.join("|", indices).replace("*", ".*")));
      scriptCode = "ctx._source.managed_index.policy.user = params.user";
    }
    if (CHANGE_POLICY_ACTION_NAME.equals(action)) {
      List<String> indices = in.readStringList();
      String policyId = in.readString();
      combinedQuery.must(new TermQueryBuilder("managed_index.change_policy.policy_id", policyId));
      combinedQuery.must(new RegexpQueryBuilder("managed_index.index",
          String.join("|", indices).replace("*", ".*")));
      scriptCode = "ctx._source.managed_index.change_policy.user = params.user; "
          + "ctx._source.managed_index.policy.user = params.user";
    }
    String finalScriptCode = scriptCode;
    return new ActionListener<>() {

      @Override
      public void onResponse(ResponseT responseT) {
        new Thread(() -> updateUserInformation(combinedQuery, finalScriptCode)).start();
        actionListener.onResponse(responseT);
      }

      @Override
      public void onFailure(Exception e) {
        actionListener.onFailure(e);
      }
    };
  }

  /**
   * Updates user settings for documents in {@value Constants#ISM_CONFIG_INDEX_NAME} index.
   *
   * @param combinedQuery query to find documents for update
   * @param scriptCode    script to execute for each found document
   */
  private void updateUserInformation(BoolQueryBuilder combinedQuery, String scriptCode) {
    SearchRequest searchRequest = new SearchRequest(ISM_CONFIG_INDEX_NAME);
    searchRequest.source().query(combinedQuery);
    log.debug("Search request is {}", searchRequest.buildDescription());

    UpdateByQueryRequest updateByQueryRequest = new UpdateByQueryRequest(ISM_CONFIG_INDEX_NAME);
    updateByQueryRequest.setQuery(combinedQuery);
    Map<String, Object> userParams = Map.of(
        "user", Map.of(
            "name", System.getenv("OPENSEARCH_USERNAME"),
            "backend_roles", List.of("admin"),
            "roles", List.of("manage_snapshots", "all_access"),
            "custom_attribute_names", List.of()
        )
    );
    Script script = new Script(ScriptType.INLINE, Script.DEFAULT_SCRIPT_LANG, scriptCode,
        userParams);
    updateByQueryRequest.setScript(script);

    for (int i = 0; i < 50; i++) {
      try {
        ActionFuture<SearchResponse> searchResponse = client.search(searchRequest);
        SearchHit[] searchHits = searchResponse.get().getHits().getHits();
        log.debug("Found by search request documents are {}", Arrays.toString(searchHits));

        if (searchHits.length > 0 && !Arrays.toString(searchHits).contains(ISM_ROLE_NAME)) {
          return;
        }

        if (searchHits.length > 0) {
          ActionFuture<BulkByScrollResponse> bulkResponseFuture = client.execute(
              UpdateByQueryAction.INSTANCE, updateByQueryRequest);
          BulkByScrollResponse bulkResponse = bulkResponseFuture.get();
          log.debug("Update by query response is {}", bulkResponse.toString());
          if (searchHits.length == bulkResponse.getUpdated()
              && bulkResponse.getVersionConflicts() == 0) {
            return;
          }
        }
      } catch (InterruptedException | ExecutionException e) {
        log.error("The problem occurred during requests performing", e);
      }
      try {
        Thread.sleep(100);
      } catch (InterruptedException e) {
        log.error("Thread is interrupted");
        return;
      }
    }
  }

  /**
   * Checks whether operation is permitted by received list of indices and policy name.
   *
   * @param indices        the list of index names
   * @param policyId       the name of policy
   * @param resourcePrefix the resource prefix
   * @return true if operation is permitted, false otherwise
   */
  @VisibleForTesting
  protected boolean allowAccess(List<String> indices, String policyId, String resourcePrefix) {
    return (indices == null || indices.stream().allMatch(index -> index.startsWith(resourcePrefix)))
        && (policyId == null || policyId.startsWith(resourcePrefix));
  }

  /**
   * Checks whether <a href="https://opensearch.org/docs/latest/im-plugin/ism/api/#add-policy">add
   * policy operation</a> is permitted for specified resource prefix.
   *
   * @param in             the input stream with request information
   * @param resourcePrefix the resource prefix
   * @return true if operation is permitted, false otherwise
   * @throws IOException if there is an issue with reading from StreamInput
   */
  private boolean allowAddPolicyAction(StreamInput in, String resourcePrefix) throws IOException {
    List<String> indices = in.readStringList();
    String policyId = in.readString();
    log.info("Add [{}] policy for {} indices", policyId, indices);
    return allowAccess(indices, policyId, resourcePrefix);
  }

  /**
   * Checks whether <a
   * href="https://opensearch.org/docs/latest/im-plugin/ism/api/#update-managed-index-policy">update
   * managed index policy operation</a> is permitted for specified resource prefix.
   *
   * @param in             the input stream with request information
   * @param resourcePrefix the resource prefix
   * @return true if operation is permitted, false otherwise
   * @throws IOException if there is an issue with reading from StreamInput
   */
  private boolean allowChangePolicyAction(StreamInput in, String resourcePrefix)
      throws IOException {
    List<String> indices = in.readStringList();
    String policyId = in.readString();
    log.info("Change [{}] policy for {} indices", policyId, indices);
    return allowAccess(indices, policyId, resourcePrefix);
  }

  /**
   * Checks whether <a
   * href="https://opensearch.org/docs/latest/im-plugin/ism/api/#delete-policy">delete policy
   * operation</a> is permitted for specified resource prefix.
   *
   * @param in             the input stream with request information
   * @param resourcePrefix the resource prefix
   * @return true if operation is permitted, false otherwise
   * @throws IOException if there is an issue with reading from StreamInput
   */
  private boolean allowDeletePolicyAction(StreamInput in, String resourcePrefix)
      throws IOException {
    String policyId = in.readString();
    log.info("Delete [{}] policy", policyId);
    return allowAccess(null, policyId, resourcePrefix);
  }

  /**
   * Checks whether <a
   * href="https://opensearch.org/docs/latest/im-plugin/ism/api/#explain-index">explain index
   * operation</a> is permitted for specified resource prefix.
   *
   * @param in             the input stream with request information
   * @param resourcePrefix the resource prefix
   * @return true if operation is permitted, false otherwise
   * @throws IOException if there is an issue with reading from StreamInput
   */
  private boolean allowExplainAction(StreamInput in, String resourcePrefix) throws IOException {
    final List<String> indices = in.readStringList();
    in.readBoolean(); // local
    in.readTimeValue(); // clusterManagerTimeout
    in.readInt(); // size of SearchParams
    in.readInt(); // from of SearchParams
    in.readString(); // sortField of SearchParams
    in.readString(); // sortOrder of SearchParams
    in.readString(); // queryString of SearchParams
    String policyId = null;
    if (in.readBoolean()) {
      policyId = in.readOptionalString();
    }
    log.info("Explain {} indices for {} policy", indices, policyId == null ? "each" : policyId);
    return allowAccess(indices, policyId, resourcePrefix);
  }

  /**
   * Checks whether <a href="https://opensearch.org/docs/latest/im-plugin/ism/api/#get-policy">get
   * policy operation</a> is permitted for specified resource prefix.
   *
   * @param in             the input stream with request information
   * @param resourcePrefix the resource prefix
   * @return true if operation is permitted, false otherwise
   * @throws IOException if there is an issue with reading from StreamInput
   */
  private boolean allowGetPolicyAction(StreamInput in, String resourcePrefix) throws IOException {
    String policyId = in.readString();
    log.info("Get [{}] policy", policyId);
    return allowAccess(null, policyId, resourcePrefix);
  }

  /**
   * Checks whether <a
   * href="https://opensearch.org/docs/latest/im-plugin/ism/api/#create-policy">create policy
   * operation</a> is permitted for specified resource prefix.
   *
   * @param request        the request information
   * @param resourcePrefix the resource prefix
   * @return true if operation is permitted, false otherwise
   * @throws IOException if there is an issue with reading from StreamInput
   */
  private boolean allowIndexPolicyAction(ActionRequest request, String resourcePrefix)
      throws IOException {
    JsonNode policyContent = getPolicyContent(request);
    String policyId = policyContent.get("policy_id").asText();
    JsonNode ismTemplate = policyContent.get("ism_template");
    log.info("Create [{}] policy with {} ISM template", policyId, ismTemplate);
    if (ismTemplate != null) {
      for (JsonNode template : ismTemplate) {
        JsonNode indexPatterns = template.get("index_patterns");
        if (indexPatterns != null && !StreamSupport.stream(indexPatterns.spliterator(), false)
            .allMatch(index -> index.asText().startsWith(resourcePrefix))) {
          return false;
        }
      }
    }
    return policyId.startsWith(resourcePrefix);
  }

  /**
   * Checks whether <a
   * href="https://opensearch.org/docs/latest/im-plugin/ism/api/#remove-policy-from-index">remove
   * policy from index operation</a> is permitted for specified resource prefix.
   *
   * @param in             the input stream with request information
   * @param resourcePrefix the resource prefix
   * @return true if operation is permitted, false otherwise
   * @throws IOException if there is an issue with reading from StreamInput
   */
  private boolean allowRemovePolicyAction(StreamInput in, String resourcePrefix)
      throws IOException {
    List<String> indices = in.readStringList();
    log.info("Remove policy from {} indices", indices);
    return allowAccess(indices, null, resourcePrefix);
  }

  /**
   * Checks whether <a
   * href="https://opensearch.org/docs/latest/im-plugin/ism/api/#retry-failed-index">retry failed
   * index operation</a> is permitted for specified resource prefix.
   *
   * @param in             the input stream with request information
   * @param resourcePrefix the resource prefix
   * @return true if operation is permitted, false otherwise
   * @throws IOException if there is an issue with reading from StreamInput
   */
  private boolean allowRetryFailedManagedIndexAction(StreamInput in, String resourcePrefix)
      throws IOException {
    List<String> indices = in.readStringList();
    log.info("Retry {} indices", indices);
    return allowAccess(indices, null, resourcePrefix);
  }

  /**
   * Returns the content of policy received from the specified request.
   *
   * @param request the request information
   * @return JSON node with policy information
   * @throws IOException if there is an issue with reading from StreamInput
   */
  @VisibleForTesting
  protected JsonNode getPolicyContent(ActionRequest request) throws IOException {
    ToXContentObject policy;
    try {
      Method getPolicy = request.getClass().getMethod("getPolicy");
      policy = (ToXContentObject) getPolicy.invoke(request);
    } catch (NoSuchMethodException | IllegalAccessException | InvocationTargetException e) {
      throw new RuntimeException(e);
    }
    BytesReference bytesReference = org.opensearch.core.xcontent.XContentHelper.toXContent(policy,
        XContentType.JSON, ToXContent.EMPTY_PARAMS, false);
    Map<String, Object> policyObject = XContentHelper.convertToMap(bytesReference, false,
        (MediaType) XContentType.JSON).v2();
    log.debug("Policy content is {}", policyObject);
    ObjectMapper mapper = new ObjectMapper();
    byte[] policyBytes = mapper.writeValueAsBytes(policyObject);
    return mapper.readTree(policyBytes).get("policy");
  }

  /**
   * Reads data from Writeable and returns them as stream.
   *
   * @param writeable object allowing to write content to a StreamOutput and read it from a
   *                  StreamInput
   * @return the input stream
   * @throws IOException if there is an issue with writing to StreamOutput
   */
  private StreamInput getStreamInput(Writeable writeable) throws IOException {
    BytesStreamOutput out = new BytesStreamOutput();
    writeable.writeTo(out);
    return out.bytes().streamInput();
  }
}
