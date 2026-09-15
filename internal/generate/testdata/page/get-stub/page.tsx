import { Text, Title } from "@mantine/core";
import { api } from "@/lib/api";

type Params = { id: string };

export default async function GetStubPage({
  params,
}: {
  params: Promise<Params>;
}) {
  const { id } = await params;
  const { data, error } = await api.example.getStub({ path: { id } });

  return (
    <>
      <Title order={1}>Get a stub by id</Title>
      {error && (
        <Text c="red" role="alert">
          {error.detail ?? "Request failed"}
        </Text>
      )}
      {data && (
        <dl>
          <dt>createdAt</dt>
          <dd>{data.createdAt}</dd>
          <dt>id</dt>
          <dd>{data.id}</dd>
          <dt>name</dt>
          <dd>{data.name}</dd>
        </dl>
      )}
    </>
  );
}
