import { Title } from "@mantine/core";
import { CreateStubForm } from "./create-stub-form";

export default function CreateStubPage() {
  return (
    <>
      <Title order={1}>Create a stub</Title>
      <CreateStubForm />
    </>
  );
}
