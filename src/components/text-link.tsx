import type {
  AnchorHTMLAttributes,
  ReactNode,
} from "react";

type TextLinkProps = Omit<
  AnchorHTMLAttributes<HTMLAnchorElement>,
  "href"
> & {
  href: string;
  children: ReactNode;
  className?: string;
};

export function TextLink({
  children,
  className = "",
  ...props
}: TextLinkProps) {
  return (
    <a
      className={`text-link inline-flex min-h-11 items-center gap-2 font-semibold ${className}`}
      {...props}
    >
      <span>{children}</span>
    </a>
  );
}
