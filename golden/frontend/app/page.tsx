import { Anchor, Button, Title } from "@mantine/core";
import { api } from "@/lib/api";
import { logout } from "./actions";

export default async function Page() {
  const { data: user } = await api.users.getCurrentUser();

  return (
    <>
      <Title order={1}>golden-app</Title>
      {user ? (
        <>
          <Title order={2}>{user.email}</Title>
          <form action={logout}>
            <Button type="submit">Log out</Button>
          </form>
        </>
      ) : (
        <Anchor href="/login">Log in</Anchor>
      )}
    </>
  );
}
