import { NextRequest, NextResponse } from 'next/server';

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8080';

export async function GET(request, { params }) {
  return proxyRequest(request, params.path);
}

export async function POST(request, { params }) {
  return proxyRequest(request, params.path);
}

export async function PUT(request, { params }) {
  return proxyRequest(request, params.path);
}

export async function DELETE(request, { params }) {
  return proxyRequest(request, params.path);
}

async function proxyRequest(request, pathSegments = []) {
  const url = new URL(`${BACKEND_URL}/auth/${pathSegments.join('/') || ''}`);
  
  // Forward query params
  url.search = request.nextUrl.search;
  
  const body = await request.text();
  
  const response = await fetch(url.toString(), {
    method: request.method,
    headers: {
      'Content-Type': 'application/json',
      // Forward auth cookies/session
      ...Object.fromEntries(request.cookies.getAll().map(cookie => [cookie.name, cookie.value])),
      // Forward other relevant headers
      ...Object.fromEntries(request.headers.entries()),
    },
    body: body || undefined,
    credentials: 'include',
  });
  
  // Proxy response headers (except hop-by-hop)
  const headers = new Headers();
  response.headers.forEach((value, key) => {
    if (!['set-cookie'].includes(key.toLowerCase())) {
      headers.append(key, value);
    }
  });
  
  // Forward Set-Cookie for session
  response.headers.getSetCookie().forEach(cookie => {
    headers.append('Set-Cookie', cookie);
  });
  
  const data = await response.text();
  
  return new NextResponse(data, {
    status: response.status,
    statusText: response.statusText,
    headers,
  });
}
