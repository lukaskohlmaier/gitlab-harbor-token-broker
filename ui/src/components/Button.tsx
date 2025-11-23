import { cn } from "../lib/utils";

interface ButtonProps extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost";
  size?: "default" | "sm" | "lg" | "icon";
}

export function Button({
  className,
  variant = "default",
  size = "default",
  ...props
}: ButtonProps) {
  return (
    <button
      className={cn(
        "inline-flex items-center justify-center rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50",
        {
          "bg-gray-900 text-gray-50 hover:bg-gray-900/90": variant === "default",
          "bg-red-500 text-gray-50 hover:bg-red-500/90": variant === "destructive",
          "border border-gray-200 bg-white hover:bg-gray-100": variant === "outline",
          "bg-gray-100 text-gray-900 hover:bg-gray-100/80": variant === "secondary",
          "hover:bg-gray-100 hover:text-gray-900": variant === "ghost",
          "h-10 px-4 py-2": size === "default",
          "h-9 rounded-md px-3": size === "sm",
          "h-11 rounded-md px-8": size === "lg",
          "h-10 w-10": size === "icon",
        },
        className
      )}
      {...props}
    />
  );
}
