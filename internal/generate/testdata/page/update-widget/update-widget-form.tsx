"use client";

import { Button, Checkbox, NumberInput, PasswordInput, Stack, Text, TextInput } from "@mantine/core";
import { useActionState } from "react";
import { updateWidget } from "./actions";

export function UpdateWidgetForm({ path }: { path: { slug: string[] } }) {
  const [state, action, pending] = useActionState(updateWidget.bind(null, path), {});

  return (
    <form action={action}>
      <Stack maw={360}>
        <NumberInput name="count" label="Count" required />
        <TextInput name="email" type="email" label="Email" required />
        <Checkbox name="enabled" label="Enabled" required />
        <TextInput name="note" label="Note" />
        <PasswordInput name="password" label="Password" required />
        <NumberInput name="price" label="Price" />
        {/* TODO: tags is not rendered (array or object) */}
        {state.error && (
          <Text c="red" role="alert">
            {state.error}
          </Text>
        )}
        {state.data && (
          <dl>
            <dt>count</dt>
            <dd>{state.data.count}</dd>
            <dt>enabled</dt>
            <dd>{String(state.data.enabled)}</dd>
            <dt>name</dt>
            <dd>{state.data.name}</dd>
            {/* TODO: tags is not a primitive; render it manually */}
          </dl>
        )}
        <Button type="submit" loading={pending}>
          Update a widget
        </Button>
      </Stack>
    </form>
  );
}
