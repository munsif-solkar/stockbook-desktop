import { AuthScreen } from "../components/AuthScreen";
import type { Company } from "../api";

export function RegisterPage({
  onAuthenticated,
}: {
  onAuthenticated: (company: Company) => void;
}) {
  return <AuthScreen formMode="register" onAuthenticated={onAuthenticated} />;
}
