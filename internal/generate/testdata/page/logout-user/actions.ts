"use server";

import { api } from "@/lib/api";

export type LogoutUserState = { done?: true; error?: string };

export async function logoutUser(
  _prev: LogoutUserState,
  _formData: FormData,
): Promise<LogoutUserState> {
  const { error } = await api.users.logoutUser();
  if (error) {
    return { error: error.detail ?? "Request failed" };
  }
  return { done: true };
}
