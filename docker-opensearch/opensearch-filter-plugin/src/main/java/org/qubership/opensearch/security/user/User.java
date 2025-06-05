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

package org.qubership.opensearch.security.user;

import java.io.IOException;
import java.util.ArrayList;
import java.util.Collections;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;
import javax.annotation.Nonnull;
import javax.annotation.Nullable;
import org.opensearch.core.common.io.stream.StreamInput;
import org.opensearch.core.common.io.stream.StreamOutput;
import org.opensearch.core.common.io.stream.Writeable;

public final class User implements Writeable {

  @Nonnull
  private final String name;
  private final Set<String> backendRoles = Collections.synchronizedSet(new HashSet<>());
  private final Set<String> roles = Collections.synchronizedSet(new HashSet<>());
  private Map<String, String> attributes = Collections.synchronizedMap(new HashMap<>());
  @Nullable
  private String requestedTenant;

  /**
   * Creates user with specified parameters.
   */
  public User(@Nonnull String name, Set<String> backendRoles, Set<String> roles,
      Map<String, String> attributes, @Nullable String requestedTenant) {
    this.name = name;
    if (backendRoles != null) {
      this.backendRoles.addAll(backendRoles);
    }
    if (roles != null) {
      this.roles.addAll(roles);
    }
    if (attributes != null) {
      this.attributes.putAll(attributes);
    }
    this.requestedTenant = requestedTenant;
  }

  /**
   * Creates user from input stream.
   */
  public User(StreamInput in) throws IOException {
    this.name = in.readString();
    this.backendRoles.addAll(in.readList(StreamInput::readString));
    this.requestedTenant = in.readString();
    if (requestedTenant.isEmpty()) {
      requestedTenant = null;
    }
    this.attributes = Collections.synchronizedMap(
        in.readMap(StreamInput::readString, StreamInput::readString));
    this.roles.addAll(in.readList(StreamInput::readString));
  }

  @Nonnull
  public String getName() {
    return name;
  }

  @Nonnull
  public Set<String> getBackendRoles() {
    return backendRoles;
  }

  @Nonnull
  public Map<String, String> getAttributes() {
    return this.attributes == null ? Collections.emptyMap() : this.attributes;
  }

  @Override
  public void writeTo(StreamOutput out) throws IOException {
    out.writeString(this.name);
    out.writeStringCollection(new ArrayList<>(this.backendRoles));
    out.writeString(this.requestedTenant == null ? "" : this.requestedTenant);
    out.writeMap(this.attributes, StreamOutput::writeString, StreamOutput::writeString);
    out.writeStringCollection(new ArrayList<>(roles));
  }

  @Override
  public String toString() {
    return "User [name=" + name + ", backend_roles=" + backendRoles
        + ", requestedTenant=" + requestedTenant + "]";
  }
}