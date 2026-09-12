import { Title } from "@mantine/core";
import { LoginForm } from "./login-form";

export default function LoginPage() {
  return (
    <>
      <Title order={1}>Log in</Title>
      <LoginForm />
    </>
  );
}
