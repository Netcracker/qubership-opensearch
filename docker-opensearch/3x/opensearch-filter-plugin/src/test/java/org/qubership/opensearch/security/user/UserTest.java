package org.qubership.opensearch.security.user;

import static org.junit.Assert.assertEquals;

import java.io.IOException;
import java.util.Map;
import java.util.Set;
import org.junit.Test;
import org.opensearch.common.io.stream.BytesStreamOutput;
import org.opensearch.core.common.io.stream.StreamInput;
import org.qubership.opensearch.Constants;

public class UserTest {

  @Test
  public void testUserCreation() throws IOException {
    User user = new User("test_user", Set.of(Constants.ISM_ROLE_NAME), null,
        Map.of(Constants.RESOURCE_PREFIX_ATTRIBUTE_NAME, "7e2df6b3-df52-49fe-8d03-fb32d9b96de0"), null);
    BytesStreamOutput out = new BytesStreamOutput();
    user.writeTo(out);
    StreamInput in = out.bytes().streamInput();
    User streamedUser = new User(in);
    assertEquals(user.getName(), streamedUser.getName());
    assertEquals(user.getBackendRoles(), streamedUser.getBackendRoles());
    assertEquals(user.getAttributes(), streamedUser.getAttributes());
  }
}
