import type {
  AnchorHTMLAttributes,
  ReactNode,
} from "react";

type ButtonLinkProps = Omit<
  AnchorHTMLAttributes<HTMLAnchorElement>,
  "href"
> & {
  href: string;
  children: ReactNode;
  variant?: "primary" | "secondary";
  className?: string;
};

export function ButtonLink({
  children,
  variant = "primary",
  className = "",
  ...props
}: ButtonLinkProps) {
  return (
    <a
      className={`button-link button-link-${variant} ${className}`}
      {...props}
    >
      {children}
    </a>
  );
}
