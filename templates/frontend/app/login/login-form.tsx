"use client";

import { Button, PasswordInput, Stack, Text, TextInput } from "@mantine/core";
import { useActionState } from "react";
import { login } from "./actions";

export function LoginForm() {
  const [state, action, pending] = useActionState(login, {});

  return (
    <form action={action}>
      <Stack maw={360}>
        <TextInput
          name="email"
          type="email"
          label="Email"
          autoComplete="email"
          required
        />
        <PasswordInput
          name="password"
          label="Password"
          autoComplete="current-password"
          required
        />
        {state.error && (
          <Text c="red" role="alert">
            {state.error}
          </Text>
        )}
        <Button type="submit" loading={pending}>
          Log in
        </Button>
      </Stack>
    </form>
  );
}
