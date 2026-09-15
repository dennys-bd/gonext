import { Title } from "@mantine/core";
import { LogoutUserForm } from "./logout-user-form";

export default function LogoutUserPage() {
  return (
    <>
      <Title order={1}>Log out</Title>
      <LogoutUserForm />
    </>
  );
}
