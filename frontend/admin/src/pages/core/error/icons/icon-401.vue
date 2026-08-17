<template>
  <svg height="600" viewBox="0 0 600 600" width="600" xmlns="http://www.w3.org/2000/svg">
    <!-- Background circle -->
    <circle cx="300" cy="300" fill="#f5f5f5" r="280" />

    <!-- Lock body -->
    <rect fill="#3f3d56" height="140" rx="16" ry="16" width="180" x="210" y="260" />

    <!-- Lock shackle -->
    <path
      d="M240 260 V200 A60 60 0 0 1 360 200 V260"
      fill="none"
      stroke="#3f3d56"
      stroke-width="24"
      stroke-linecap="round"
    />

    <!-- Keyhole -->
    <circle cx="300" cy="320" fill="#fff" r="20" />
    <rect fill="#fff" height="40" width="16" x="292" y="320" />

    <!-- Warning triangle -->
    <path d="M280 140 L320 140 L300 100 Z" fill="#ff6b6b" />

    <!-- Exclamation mark -->
    <rect fill="#fff" height="20" width="6" x="297" y="112" />
    <circle cx="300" cy="148" fill="#fff" r="4" />

    <!-- Decorative elements -->
    <circle cx="150" cy="200" fill="#e0e0e0" r="8" />
    <circle cx="450" cy="200" fill="#e0e0e0" r="8" />
    <circle cx="150" cy="400" fill="#e0e0e0" r="8" />
    <circle cx="450" cy="400" fill="#e0e0e0" r="8" />

    <!-- Shield icon -->
    <path
      d="M420 420 L460 400 L460 450 C460 480 440 500 420 510 C400 500 380 480 380 450 L380 400 Z"
      fill="#a0616a"
    />
    <path
      d="M420 435 L435 425 L435 445 C435 455 428 462 420 466 C412 462 405 455 405 445 L405 425 Z"
      fill="#fff"
    />
  </svg>
</template>
