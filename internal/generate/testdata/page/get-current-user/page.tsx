import { Text, Title } from "@mantine/core";
import { api } from "@/lib/api";

export default async function GetCurrentUserPage() {
  const { data, error } = await api.users.getCurrentUser();

  return (
    <>
      <Title order={1}>Get the current user</Title>
      {error && (
        <Text c="red" role="alert">
          {error.detail ?? "Request failed"}
        </Text>
      )}
      {data && (
        <dl>
          <dt>email</dt>
          <dd>{data.email}</dd>
          <dt>emailVerified</dt>
          <dd>{String(data.emailVerified)}</dd>
          <dt>id</dt>
          <dd>{data.id}</dd>
          {/* TODO: permissions is not a primitive; render it manually */}
          <dt>role</dt>
          <dd>{data.role}</dd>
        </dl>
      )}
    </>
  );
}
