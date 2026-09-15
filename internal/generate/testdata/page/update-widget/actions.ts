"use server";

import { api } from "@/lib/api";

type Data = Awaited<ReturnType<typeof api.example.updateWidget>>["data"];

export type UpdateWidgetState = { done?: true; data?: Data; error?: string };

export async function updateWidget(
  path: { slug: string[] },
  _prev: UpdateWidgetState,
  formData: FormData,
): Promise<UpdateWidgetState> {
  const { slug } = path;
  const { data, error } = await api.example.updateWidget({
    path: { slug: slug.join("/") },
    body: {
      count: Number(formData.get("count")),
      email: String(formData.get("email") ?? ""),
      enabled: formData.get("enabled") === "on",
      note: String(formData.get("note") ?? "") || undefined,
      password: String(formData.get("password") ?? ""),
      price: formData.get("price") ? Number(formData.get("price")) : undefined,
      tags: [],
    },
  });
  if (error) {
    return { error: error.detail ?? "Request failed" };
  }
  return { done: true, data };
}
