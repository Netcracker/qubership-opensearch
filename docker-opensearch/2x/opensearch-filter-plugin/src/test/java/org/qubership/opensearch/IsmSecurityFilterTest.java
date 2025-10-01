package org.qubership.opensearch;

import static org.junit.Assert.assertFalse;
import static org.junit.Assert.assertTrue;
import static org.mockito.Mockito.mock;

import com.fasterxml.jackson.databind.ObjectMapper;
import java.io.IOException;
import java.util.List;
import java.util.Map;
import org.junit.BeforeClass;
import org.junit.Test;
import org.mockito.Mockito;
import org.opensearch.action.ActionRequest;
import org.opensearch.action.ActionRequestValidationException;
import org.opensearch.client.Client;
import org.opensearch.common.unit.TimeValue;
import org.opensearch.core.common.io.stream.StreamOutput;
import org.opensearch.threadpool.ThreadPool;

public class IsmSecurityFilterTest {

  private static final String RESOURCE_PREFIX = "test";
  private static final String PERMISSIBLE_POLICY_NAME = "test_policy";
  private static final String IMPERMISSIBLE_POLICY_NAME = "tmp_policy";

  private static final ObjectMapper OBJECT_MAPPER = new ObjectMapper();
  private static IsmSecurityFilter filter;

  /**
   * The method to be run before any of the test methods in the class.
   */
  @BeforeClass
  public static void beforeClass() {
    filter = Mockito.spy(new IsmSecurityFilter(mock(ThreadPool.class), mock(Client.class)));
  }

  @Test
  public void testAllowAccessForPermissiblePolicy() {
    boolean isAllowed = filter.allowAccess(null, PERMISSIBLE_POLICY_NAME, RESOURCE_PREFIX);
    assertTrue("Access should be granted for permissible policy", isAllowed);
  }

  @Test
  public void testAllowAccessForImpermissiblePolicy() {
    boolean isAllowed = filter.allowAccess(null, IMPERMISSIBLE_POLICY_NAME, RESOURCE_PREFIX);
    assertFalse("Access should not be granted for impermissible policy", isAllowed);
  }

  @Test
  public void testAllowAccessForPermissibleIndices() {
    List<String> indices = List.of("test-000001", "testing-000001");
    boolean isAllowed = filter.allowAccess(indices, null, RESOURCE_PREFIX);
    assertTrue("Access should be granted for permissible indices", isAllowed);
  }

  @Test
  public void testAllowAccessForImpermissibleIndices() {
    List<String> indices = List.of("tmp-000001", "tes-000001");
    boolean isAllowed = filter.allowAccess(indices, null, RESOURCE_PREFIX);
    assertFalse("Access should not be granted for impermissible indices", isAllowed);
  }

  @Test
  public void testAllowAccessForPermissibleAndImpermissibleIndices() {
    List<String> indices = List.of("tmp-000001", "test-000001", "tes-000001");
    boolean isAllowed = filter.allowAccess(indices, null, RESOURCE_PREFIX);
    assertFalse("Access should not be granted for impermissible indices", isAllowed);
  }

  @Test
  public void testAllowAccessForPermissiblePolicyAndIndices() {
    List<String> indices = List.of("test-000001");
    boolean isAllowed = filter.allowAccess(indices, PERMISSIBLE_POLICY_NAME, RESOURCE_PREFIX);
    assertTrue("Access should be granted for permissible policy and indices", isAllowed);
  }

  @Test
  public void testAllowAccessForImpermissiblePolicyAndPermissibleIndices() {
    List<String> indices = List.of("test-000001", "testing-000001");
    boolean isAllowed = filter.allowAccess(indices, IMPERMISSIBLE_POLICY_NAME, RESOURCE_PREFIX);
    assertFalse(
        "Access should not be granted for impermissible policy" + " and permissible indices",
        isAllowed);
  }

  @Test
  public void testAllowAccessForPermissiblePolicyAndImpermissibleIndices() {
    List<String> indices = List.of("tmp-000001", "tes-000001", "test-000001");
    boolean isAllowed = filter.allowAccess(indices, PERMISSIBLE_POLICY_NAME, RESOURCE_PREFIX);
    assertFalse(
        "Access should not be granted for permissible policy" + " and impermissible indices",
        isAllowed);
  }

  @Test
  public void testAllowAccessForImpermissiblePolicyAndIndices() {
    List<String> indices = List.of("test-000001", "tes-000001");
    boolean isAllowed = filter.allowAccess(indices, IMPERMISSIBLE_POLICY_NAME, RESOURCE_PREFIX);
    assertFalse("Access should not be granted for impermissible policy and indices", isAllowed);
  }

  @Test
  public void testAllowAddPolicyWithPermissibleNameAndIndices() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeStringCollection(List.of("test-000001", "testing-000001"));
        out.writeString(PERMISSIBLE_POLICY_NAME);
        out.writeString("_default");
      }
    };
    boolean isAllowed = filter.allowAction(Constants.ADD_POLICY_ACTION_NAME, request, RESOURCE_PREFIX);
    assertTrue("Adding policy with permissible name and indices should be allowed", isAllowed);
  }

  @Test
  public void testAllowAddPolicyWithPermissibleNameAndImpermissibleIndex() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeStringCollection(List.of("test-000001", "tmp-000001"));
        out.writeString(PERMISSIBLE_POLICY_NAME);
        out.writeString("_default");
      }
    };
    boolean isAllowed = filter.allowAction(Constants.ADD_POLICY_ACTION_NAME, request, RESOURCE_PREFIX);
    assertFalse(
        "Adding policy with permissible name and impermissible index " + "should not be allowed",
        isAllowed);
  }

  @Test
  public void testAllowChangePolicyWithPermissibleNameAndIndices() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeStringCollection(List.of("test-000001", "testing-000001"));
        out.writeString(PERMISSIBLE_POLICY_NAME);
        out.writeOptionalString("test");
      }
    };
    boolean isAllowed = filter.allowAction(Constants.CHANGE_POLICY_ACTION_NAME, request, RESOURCE_PREFIX);
    assertTrue("Changing policy with permissible name and indices should be allowed", isAllowed);
  }

  @Test
  public void testAllowChangePolicyWithImpermissibleNameAndIndices() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeStringCollection(List.of("tes-000001", "tmp-000001"));
        out.writeString(IMPERMISSIBLE_POLICY_NAME);
        out.writeOptionalString("test");
      }
    };
    boolean isAllowed = filter.allowAction(Constants.CHANGE_POLICY_ACTION_NAME, request, RESOURCE_PREFIX);
    assertFalse("Changing policy with impermissible name and indices should not be allowed",
        isAllowed);
  }

  @Test
  public void testDeletePolicyWithPermissibleName() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeString(PERMISSIBLE_POLICY_NAME);
      }
    };
    boolean isAllowed = filter.allowAction(Constants.DELETE_POLICY_ACTION_NAME, request, RESOURCE_PREFIX);
    assertTrue("Removing policy with permissible name should be allowed", isAllowed);
  }

  @Test
  public void testDeletePolicyWithImpermissibleName() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeString(IMPERMISSIBLE_POLICY_NAME);
      }
    };
    boolean isAllowed = filter.allowAction(Constants.DELETE_POLICY_ACTION_NAME, request, RESOURCE_PREFIX);
    assertFalse("Removing policy with impermissible name should not be allowed", isAllowed);
  }

  @Test
  public void testAllowExplainIndexWithoutFilter() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeStringCollection(List.of("test-000001", "tests-000001"));
        out.writeBoolean(true); // local
        out.writeTimeValue(new TimeValue(1000)); // clusterManagerTimeout
        out.writeInt(20); // size of SearchParams
        out.writeInt(0); // from of SearchParams
        out.writeString("test"); // sortField of SearchParams
        out.writeString("asc"); // sortOrder of SearchParams
        out.writeString("*"); // queryString of SearchParams
        out.writeBoolean(false); // ExplainFilter
      }
    };
    boolean isAllowed = filter.allowAction(Constants.EXPLAIN_ACTION_NAME, request, RESOURCE_PREFIX);
    assertTrue("Explaining index with permissible name should not be allowed", isAllowed);
  }

  @Test
  public void testAllowExplainIndexWithImpermissiblePolicyInFilter() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeStringCollection(List.of("test-000001", "tests-000001"));
        out.writeBoolean(true); // local
        out.writeTimeValue(new TimeValue(1000)); // clusterManagerTimeout
        out.writeInt(20); // size of SearchParams
        out.writeInt(0); // from of SearchParams
        out.writeString("test"); // sortField of SearchParams
        out.writeString("asc"); // sortOrder of SearchParams
        out.writeString("*"); // queryString of SearchParams
        out.writeBoolean(true); // ExplainFilter
        out.writeOptionalString(IMPERMISSIBLE_POLICY_NAME);
      }
    };
    boolean isAllowed = filter.allowAction(Constants.EXPLAIN_ACTION_NAME, request, RESOURCE_PREFIX);
    assertFalse("Explaining index with permissible name and impermissible policy"
        + " in ExplainFilter should not be allowed", isAllowed);
  }

  @Test
  public void testAllowGetPolicyWithPermissibleName() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeString(PERMISSIBLE_POLICY_NAME);
      }
    };
    boolean isAllowed = filter.allowAction(Constants.GET_POLICY_ACTION_NAME, request, RESOURCE_PREFIX);
    assertTrue("Getting policy with permissible name should be allowed", isAllowed);
  }

  @Test
  public void testAllowGetPolicyWithImpermissibleName() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeString(IMPERMISSIBLE_POLICY_NAME);
      }
    };
    boolean isAllowed = filter.allowAction(Constants.GET_POLICY_ACTION_NAME, request, RESOURCE_PREFIX);
    assertFalse("Getting policy with impermissible name should not be allowed", isAllowed);
  }

  @Test
  public void testAllowGetPolicies() throws IOException {
    ActionRequest request = new TestsActionRequest();
    boolean isAllowed = filter.allowAction(Constants.GET_POLICIES_ACTION_NAME, request, RESOURCE_PREFIX);
    assertFalse("Getting policies should not be allowed", isAllowed);
  }

  @Test
  public void testAllowRemovePolicyFromIndexWithPermissibleName() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeStringCollection(List.of("tests-000001"));
        out.writeString("_default");
      }
    };
    boolean isAllowed = filter.allowAction(Constants.REMOVE_POLICY_ACTION_NAME, request, RESOURCE_PREFIX);
    assertTrue("Removing policy from index with permissible name should be allowed", isAllowed);
  }

  @Test
  public void testAllowRemovePolicyFromIndicesWithImpermissibleNames() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeStringCollection(List.of("cat-000001", "tmp-000001", "temp-000001"));
        out.writeString("_default");
      }
    };
    boolean isAllowed = filter.allowAction(Constants.REMOVE_POLICY_ACTION_NAME, request, RESOURCE_PREFIX);
    assertFalse("Removing policy from indices with impermissible names should not be allowed",
        isAllowed);
  }

  @Test
  public void testAllowRetryFailedManagedIndexWithPermissibleName() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeStringCollection(List.of("testing-000001"));
      }
    };
    boolean isAllowed = filter.allowAction(Constants.RETRY_FAILED_MANAGED_INDEX_ACTION_NAME, request,
        RESOURCE_PREFIX);
    assertTrue("Retrying failed managed indices with permissible name should be allowed",
        isAllowed);
  }

  @Test
  public void testAllowRetryFailedManagedIndexWithImpermissibleName() throws IOException {
    ActionRequest request = new TestsActionRequest() {
      @Override
      public void writeTo(StreamOutput out) throws IOException {
        out.writeStringCollection(List.of("prom-000001", "test-000001", "last-000001"));
      }
    };
    boolean isAllowed = filter.allowAction(Constants.RETRY_FAILED_MANAGED_INDEX_ACTION_NAME, request,
        RESOURCE_PREFIX);
    assertFalse("Retrying failed managed indices with impermissible name should not be allowed",
        isAllowed);
  }

  @Test
  public void testAllowCreatePolicyWithPermissibleNameAndNoTemplate() throws IOException {
    Map<String, Object> policy =
        Map.of("policy", Map.of("policy_id", PERMISSIBLE_POLICY_NAME));
    byte[] policyBytes = OBJECT_MAPPER.writeValueAsBytes(policy);
    ActionRequest actionRequest = mock(ActionRequest.class);
    Mockito.doReturn(OBJECT_MAPPER.readTree(policyBytes).get("policy")).when(filter)
        .getPolicyContent(actionRequest);

    boolean isAllowed = filter.allowAction(Constants.INDEX_POLICY_ACTION_NAME, actionRequest,
        RESOURCE_PREFIX);
    assertTrue("Policy creation with permissible name should be allowed", isAllowed);
  }

  @Test
  public void testAllowCreatePolicyWithImpermissibleNameAndNoTemplate() throws IOException {
    Map<String, Object> policy = Map.of("policy", Map.of("policy_id", IMPERMISSIBLE_POLICY_NAME));
    byte[] policyBytes = OBJECT_MAPPER.writeValueAsBytes(policy);
    ActionRequest actionRequest = mock(ActionRequest.class);
    Mockito.doReturn(OBJECT_MAPPER.readTree(policyBytes).get("policy")).when(filter)
        .getPolicyContent(actionRequest);

    boolean isAllowed = filter.allowAction(Constants.INDEX_POLICY_ACTION_NAME, actionRequest,
        RESOURCE_PREFIX);
    assertFalse("Policy creation with impermissible name should not be allowed", isAllowed);
  }

  @Test
  public void testAllowCreatePolicyWithPermissibleNameAndTemplate() throws IOException {
    Map<String, Object> policy = Map.of("policy", Map.of("policy_id", PERMISSIBLE_POLICY_NAME,
        "ism_template", Map.of("index_patterns", List.of("test*"))));
    byte[] policyBytes = OBJECT_MAPPER.writeValueAsBytes(policy);
    ActionRequest actionRequest = mock(ActionRequest.class);
    Mockito.doReturn(OBJECT_MAPPER.readTree(policyBytes).get("policy")).when(filter)
        .getPolicyContent(actionRequest);

    boolean isAllowed = filter.allowAction(Constants.INDEX_POLICY_ACTION_NAME, actionRequest,
        RESOURCE_PREFIX);
    assertTrue("Policy creation with permissible name and ISM template should be allowed",
        isAllowed);
  }

  @Test
  public void testAllowCreatePolicyWithPermissibleNameAndImpermissibleTemplate()
      throws IOException {
    Map<String, Object> policy = Map.of("policy", Map.of("policy_id", PERMISSIBLE_POLICY_NAME,
        "ism_template", List.of(Map.of("index_patterns", List.of("tmp*", "test*")))));
    byte[] policyBytes = OBJECT_MAPPER.writeValueAsBytes(policy);
    ActionRequest actionRequest = mock(ActionRequest.class);
    Mockito.doReturn(OBJECT_MAPPER.readTree(policyBytes).get("policy")).when(filter)
        .getPolicyContent(actionRequest);

    boolean isAllowed = filter.allowAction(Constants.INDEX_POLICY_ACTION_NAME, actionRequest,
        RESOURCE_PREFIX);
    assertFalse("Policy creation with permissible name and impermissible ISM template "
        + "should not be allowed", isAllowed);
  }

  @Test
  public void testAllowCreatePolicyWithPermissibleNameAndDifferentTemplates() throws IOException {
    Map<String, Object> policy = Map.of("policy", Map.of("policy_id", PERMISSIBLE_POLICY_NAME,
        "ism_template", List.of(Map.of("index_patterns", List.of("test*")),
            Map.of("index_patterns", List.of("tmp*")))));
    byte[] policyBytes = OBJECT_MAPPER.writeValueAsBytes(policy);
    ActionRequest actionRequest = mock(ActionRequest.class);
    Mockito.doReturn(OBJECT_MAPPER.readTree(policyBytes).get("policy")).when(filter)
        .getPolicyContent(actionRequest);

    boolean isAllowed = filter.allowAction(Constants.INDEX_POLICY_ACTION_NAME, actionRequest,
        RESOURCE_PREFIX);
    assertFalse("Policy creation with permissible name and impermissible ISM template "
        + "should not be allowed", isAllowed);
  }

  @Test
  public void testAllowCreatePolicyWithImpermissibleNameAndPermissibleTemplate()
      throws IOException {
    Map<String, Object> policy = Map.of("policy", Map.of("policy_id", IMPERMISSIBLE_POLICY_NAME,
        "ism_template", List.of(Map.of("index_patterns", List.of("test*")))));
    byte[] policyBytes = OBJECT_MAPPER.writeValueAsBytes(policy);
    ActionRequest actionRequest = mock(ActionRequest.class);
    Mockito.doReturn(OBJECT_MAPPER.readTree(policyBytes).get("policy")).when(filter)
        .getPolicyContent(actionRequest);

    boolean isAllowed = filter.allowAction(Constants.INDEX_POLICY_ACTION_NAME, actionRequest,
        RESOURCE_PREFIX);
    assertFalse("Policy creation with impermissible name and permissible ISM template "
        + "should not be allowed", isAllowed);
  }

  static class TestsActionRequest extends ActionRequest {

    @Override
    public ActionRequestValidationException validate() {
      return null;
    }
  }
}
