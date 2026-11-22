import * as React from 'react';
import { cn } from '@/lib/utils';

function Skeleton({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        'animate-pulse rounded-md bg-emerald-500/10 border border-emerald-500/10',
        className
      )}
      {...props}
    />
  );
}

export { Skeleton };
