import { AuthScreen } from "../components/AuthScreen";
import type { Company } from "../api";

export function LoginPage({
  onAuthenticated,
}: {
  onAuthenticated: (company: Company) => void;
}) {
  return <AuthScreen formMode="login" onAuthenticated={onAuthenticated} />;
}
