"use server";

import { api } from "@/lib/api";

type Data = Awaited<ReturnType<typeof api.example.createStub>>["data"];

export type CreateStubState = { done?: true; data?: Data; error?: string };

export async function createStub(
  _prev: CreateStubState,
  formData: FormData,
): Promise<CreateStubState> {
  const { data, error } = await api.example.createStub({
    body: {
      name: String(formData.get("name") ?? ""),
    },
  });
  if (error) {
    return { error: error.detail ?? "Request failed" };
  }
  return { done: true, data };
}
