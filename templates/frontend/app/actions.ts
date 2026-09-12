"use server";

import { redirect } from "next/navigation";
import { api } from "@/lib/api";

// Revokes the session server-side; the relay applies Go's clearing
// Set-Cookie to the browser. A failed logout (session already gone) still
// ends on the login page, which is the state the user asked for.
export async function logout(): Promise<void> {
  await api.users.logoutUser();
  redirect("/login");
}
