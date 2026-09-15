import { Title } from "@mantine/core";
import { UpdateWidgetForm } from "./update-widget-form";

type Params = { slug: string[]; rest: string[] | undefined };

export default async function UpdateWidgetPage({
  params,
}: {
  params: Promise<Params>;
}) {
  const { slug } = await params;

  return (
    <>
      <Title order={1}>Update a widget</Title>
      <UpdateWidgetForm path={{ slug }} />
    </>
  );
}
