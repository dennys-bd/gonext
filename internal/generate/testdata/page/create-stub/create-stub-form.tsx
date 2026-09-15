"use client";

import { Button, Stack, Text, TextInput } from "@mantine/core";
import { useActionState } from "react";
import { createStub } from "./actions";

export function CreateStubForm() {
  const [state, action, pending] = useActionState(createStub, {});

  return (
    <form action={action}>
      <Stack maw={360}>
        <TextInput name="name" label="Name" required />
        {state.error && (
          <Text c="red" role="alert">
            {state.error}
          </Text>
        )}
        {state.data && (
          <dl>
            <dt>createdAt</dt>
            <dd>{state.data.createdAt}</dd>
            <dt>id</dt>
            <dd>{state.data.id}</dd>
            <dt>name</dt>
            <dd>{state.data.name}</dd>
          </dl>
        )}
        <Button type="submit" loading={pending}>
          Create a stub
        </Button>
      </Stack>
    </form>
  );
}
