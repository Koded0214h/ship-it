# typed: false
# frozen_string_literal: true

class Ship < Formula
  desc "Deploy to your VPS using plain English. AI-generated Docker, Nginx, SSL, and CI/CD."
  homepage "https://github.com/Koded0214h/ship-it"
  version "0.1.0"
  license "MIT"

  on_macos do
    on_intel do
      url "https://github.com/Koded0214h/ship-it/releases/download/v0.1.0/ship_0.1.0_darwin_amd64.tar.gz"
      sha256 "94d79bd7fb89525db0ae60c911468a6dd57b3684a6bf0689bcdb8b0d896fb164"
    end

    on_arm do
      url "https://github.com/Koded0214h/ship-it/releases/download/v0.1.0/ship_0.1.0_darwin_arm64.tar.gz"
      sha256 "8396c4e2f7ec837481cae3e0af6c1c4c418cd3288de97266698ef8f1b6bce515"
    end
  end

  on_linux do
    on_intel do
      url "https://github.com/Koded0214h/ship-it/releases/download/v0.1.0/ship_0.1.0_linux_amd64.tar.gz"
      sha256 "94a83dfbb6cf6e0125a27d81d5b9cdc9997cdd259bc0046c348869d0f68d1127"
    end

    on_arm do
      url "https://github.com/Koded0214h/ship-it/releases/download/v0.1.0/ship_0.1.0_linux_arm64.tar.gz"
      sha256 "306ac606105e53a156ea404c460b288df08ea7c604d7b8d950e4917cc75aae44"
    end
  end

  def install
    bin.install "ship"
  end

  test do
    system "#{bin}/ship", "--version"
  end
end
