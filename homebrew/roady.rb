# Typed as: Homebrew formula
# This is a template - actual formula should be in a separate homebrew-tap repository
# See: https://docs.brew.sh/Formula-Cookbook

class Roady < Formula
  desc "Planning-first system of record for software work"
  homepage "https://github.com/felixgeelhaar/roady"
  url "https://github.com/felixgeelhaar/roady.git"
  version "0.8.0"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build", "-o", bin/"roady", "./cmd/roady"
  end

  test do
    system "#{bin}/roady", "--version"
  end
end
