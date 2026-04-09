import { usePageContext } from "vike-react/usePageContext";

export default function Page() {
  const { is404 } = usePageContext();
  if (is404) {
    return (
      <>
        <p className="text-center text-xl">This page could not be found.</p>
      </>
    );
  }
  return (
    <>
      <p className="text-center text-xl">Something went wrong.</p>
    </>
  );
}
