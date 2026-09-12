"use server";

import { redirect } from "next/navigation";
import { api } from "@/lib/api";

export type LoginState = { error?: string };

const GENERIC_ERROR = "Could not log in";

// Form action for LoginForm. On success the relay has already applied Go's
// Set-Cookie to the browser, so the redirect lands on a signed-in home page.
export async function login(
  _prev: LoginState,
  formData: FormData,
): Promise<LoginState> {
  const { error } = await api.users.loginUser({
    body: {
      email: String(formData.get("email") ?? ""),
      password: String(formData.get("password") ?? ""),
    },
  });
  if (error) {
    return { error: error.detail ?? GENERIC_ERROR };
  }
  redirect("/");
}
