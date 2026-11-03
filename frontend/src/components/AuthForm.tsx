import { useState, type FormEvent } from "react";
import { api, type Company } from "../api";
import { Link } from "../router";
import { SubmitButton } from "./SubmitButton";

export function AuthForm({
  mode,
  onAuthenticated,
}: {
  mode: "login" | "register";
  onAuthenticated: (company: Company) => void;
}) {
  const isRegister = mode === "register";
  const [companyName, setCompanyName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<
    Record<string, string[] | undefined>
  >({});
  const [pending, setPending] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setPending(true);
    setError("");
    setFieldErrors({});
    try {
      const result = isRegister
        ? await api.Register({ companyName, email, password, confirmPassword })
        : await api.Login({ email, password });
      if (result.error) {
        setError(result.error);
        return;
      }
      if (result.fieldErrors) {
        setFieldErrors(result.fieldErrors);
        return;
      }
      if (result.company) {
        setPassword("");
        setConfirmPassword("");
        onAuthenticated(result.company);
      }
    } catch (submitError) {
      setError(
        submitError instanceof Error ? submitError.message : String(submitError),
      );
    } finally {
      setPending(false);
    }
  }

  return (
    <form onSubmit={submit} noValidate>
      {error ? (
        <div className="form-error" role="alert">
          {error}
        </div>
      ) : null}

      {isRegister ? (
        <div className="field">
          <label htmlFor="companyName">Company name</label>
          <input
            className="input"
            id="companyName"
            name="companyName"
            type="text"
            autoComplete="organization"
            value={companyName}
            onChange={(event) => setCompanyName(event.target.value)}
            aria-invalid={Boolean(fieldErrors.companyName)}
            required
          />
          {fieldErrors.companyName ? (
            <span className="error-text">{fieldErrors.companyName[0]}</span>
          ) : null}
        </div>
      ) : null}

      <div className="field">
        <label htmlFor="email">Email</label>
        <input
          className="input"
          id="email"
          name="email"
          type="email"
          inputMode="email"
          autoComplete="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          aria-invalid={Boolean(fieldErrors.email)}
          required
        />
        {fieldErrors.email ? (
          <span className="error-text">{fieldErrors.email[0]}</span>
        ) : null}
      </div>

      <div className="field">
        <label htmlFor="password">Password</label>
        <input
          className="input"
          id="password"
          name="password"
          type="password"
          autoComplete={isRegister ? "new-password" : "current-password"}
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          aria-invalid={Boolean(fieldErrors.password)}
          required
        />
        {fieldErrors.password ? (
          <span className="error-text">{fieldErrors.password[0]}</span>
        ) : isRegister ? (
          <span className="hint">At least 8 characters, with a letter and a number.</span>
        ) : null}
      </div>

      {isRegister ? (
        <div className="field">
          <label htmlFor="confirmPassword">Confirm password</label>
          <input
            className="input"
            id="confirmPassword"
            name="confirmPassword"
            type="password"
            autoComplete="new-password"
            value={confirmPassword}
            onChange={(event) => setConfirmPassword(event.target.value)}
            aria-invalid={Boolean(fieldErrors.confirmPassword)}
            required
          />
          {fieldErrors.confirmPassword ? (
            <span className="error-text">{fieldErrors.confirmPassword[0]}</span>
          ) : null}
        </div>
      ) : null}

      <div className="form-actions">
        <SubmitButton
          className="btn btn-primary btn-block"
          pending={pending}
          pendingLabel={isRegister ? "Creating workspace…" : "Signing in…"}
        >
          {isRegister ? "Create workspace" : "Sign in"}
        </SubmitButton>
      </div>

      <div className="form-foot">
        {isRegister ? (
          <>
            Already have an account? <Link to="/login">Sign in</Link>
          </>
        ) : (
          <>
            New here? <Link to="/register">Create an account</Link>
          </>
        )}
      </div>
    </form>
  );
}
