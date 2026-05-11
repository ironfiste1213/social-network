'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';

// /create just redirects to the feed where the create post box lives
export default function CreatePage() {
  const router = useRouter();
  useEffect(() => { router.replace('/feed'); }, [router]);
  return null;
}
