package org.qubership.opensearch.security.user;

import java.util.*;
import javax.annotation.Nonnull;
import javax.annotation.Nullable;
import java.io.IOException;
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

    public User(@Nonnull String name,
                Set<String> backendRoles,
                Set<String> roles,
                Map<String, String> attributes,
                @Nullable String requestedTenant) {
        this.name = name;
        if (backendRoles != null) this.backendRoles.addAll(backendRoles);
        if (roles != null) this.roles.addAll(roles);
        if (attributes != null) this.attributes.putAll(attributes);
        this.requestedTenant = requestedTenant;
    }

    public User(StreamInput in) throws IOException {
        this.name = in.readString();
        this.backendRoles.addAll(in.readList(StreamInput::readString));
        String tenant = in.readOptionalString();
        this.requestedTenant = (tenant == null || tenant.isEmpty()) ? null : tenant;
        this.attributes.putAll(in.readMap(StreamInput::readString, StreamInput::readString));
        this.roles.addAll(in.readList(StreamInput::readString));
    }

    @Override
    public void writeTo(StreamOutput out) throws IOException {
        out.writeString(name);
        out.writeStringCollection(backendRoles);
        out.writeOptionalString(requestedTenant);
        out.writeMap(attributes, StreamOutput::writeString, StreamOutput::writeString);
        out.writeStringCollection(roles);
    }

    public String getName() { return name; }
    public Set<String> getBackendRoles() { return backendRoles; }
    public Set<String> getRoles() { return roles; }
    public Map<String, String> getAttributes() { return attributes; }
    public String getRequestedTenant() { return requestedTenant; }

    @Override
    public String toString() {
        return "User[" + name + ", roles=" + roles + ", backendRoles=" + backendRoles + "]";
    }
}
