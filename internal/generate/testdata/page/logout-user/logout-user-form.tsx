"use client";

import { Button, Stack, Text } from "@mantine/core";
import { useActionState } from "react";
import { logoutUser } from "./actions";

export function LogoutUserForm() {
  const [state, action, pending] = useActionState(logoutUser, {});

  return (
    <form action={action}>
      <Stack maw={360}>
        {state.error && (
          <Text c="red" role="alert">
            {state.error}
          </Text>
        )}
        {state.done && <Text>Done</Text>}
        <Button type="submit" loading={pending}>
          Log out
        </Button>
      </Stack>
    </form>
  );
}
