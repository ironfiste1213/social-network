import { NextResponse } from 'next/server';

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8080';

export async function GET(request, context) {
  const { path } = await context.params;
  return proxyRequest(request, path);
}

export async function POST(request, context) {
  const { path } = await context.params;
  return proxyRequest(request, path);
}

export async function PUT(request, context) {
  const { path } = await context.params;
  return proxyRequest(request, path);
}

export async function PATCH(request, context) {
  const { path } = await context.params;
  return proxyRequest(request, path);
}

export async function DELETE(request, context) {
  const { path } = await context.params;
  return proxyRequest(request, path);
}

async function proxyRequest(request, pathSegments = []) {
  const path = pathSegments.join('/');
  const url = new URL(`${BACKEND_URL}/${path}`);
  url.search = request.nextUrl.search;

  const contentType = request.headers.get('content-type') || '';
  const isMultipart = contentType.includes('multipart/form-data');

  let body;
  const headers = {
    // Forward cookies as Cookie header
    cookie: request.cookies.getAll().map(c => `${c.name}=${c.value}`).join('; '),
  };

  if (isMultipart) {
    // Forward raw form data without content-type override
    body = await request.blob();
    headers['content-type'] = contentType;
  } else {
    const text = await request.text();
    body = text || undefined;
    if (body) headers['content-type'] = 'application/json';
  }

  const response = await fetch(url.toString(), {
    method: request.method,
    headers,
    body,
  });

  // Forward response headers
  const responseHeaders = new Headers();
  response.headers.forEach((value, key) => {
    if (key.toLowerCase() !== 'set-cookie') {
      responseHeaders.append(key, value);
    }
  });
  response.headers.getSetCookie?.().forEach(cookie => {
    responseHeaders.append('Set-Cookie', cookie);
  });

  const data = await response.arrayBuffer();
  return new NextResponse(data, {
    status: response.status,
    statusText: response.statusText,
    headers: responseHeaders,
  });
}
