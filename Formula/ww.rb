class Ww < Formula
  desc "Fast worktree switching for safer parallel work"
  homepage "https://github.com/unix2dos/ww"
  version "0.7.0"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/unix2dos/ww/releases/download/v0.7.0/ww-v0.7.0-darwin-arm64.tar.gz"
    sha256 "294c52b2d508be5b5f8371dc547a578a812360cbee8ac993fec247a3a735b62e"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/unix2dos/ww/releases/download/v0.7.0/ww-v0.7.0-darwin-amd64.tar.gz"
    sha256 "3ce8fe8d6fc060ee7f239071174c8abb8868040b94f4fc8e182528ae82f6f86a"
  elsif OS.linux? && Hardware::CPU.arm?
    url "https://github.com/unix2dos/ww/releases/download/v0.7.0/ww-v0.7.0-linux-arm64.tar.gz"
    sha256 "0dd37a1d8168b729d769865f91022a56a64dbfed0bb75a4215c7005adf7e6b0d"
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/unix2dos/ww/releases/download/v0.7.0/ww-v0.7.0-linux-amd64.tar.gz"
    sha256 "e75741a2f8bb76be7deb002362bd93465065d0ba69f56102ca37622f9c346133"
  end

  def install
    bin.install "bin/ww-helper"
    libexec.install "shell/ww.sh"
    doc.install "README.md"
  end

  def caveats
    <<~EOS
      `ww` changes the current shell directory, so Homebrew installs the helper and shell library
      but leaves shell activation to you.

      Add one line to your shell rc file:

      For zsh:
        eval "$("#{opt_bin}/ww-helper" init zsh)"

      For bash:
        eval "$("#{opt_bin}/ww-helper" init bash)"
    EOS
  end

  test do
    assert_path_exists libexec/"ww.sh"
    assert_match "Usage: ww-helper", shell_output("#{bin}/ww-helper help")
  end
end
