class Ww < Formula
  desc "Fast worktree switching for safer parallel work"
  homepage "https://github.com/unix2dos/ww"
  version "0.6.0"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/unix2dos/ww/releases/download/v0.6.0/ww-v0.6.0-darwin-arm64.tar.gz"
    sha256 "9aa0d201ea984b4e06348fc03b7ec6bf2337ea9ff79f3016369f7c0321f0e09a"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/unix2dos/ww/releases/download/v0.6.0/ww-v0.6.0-darwin-amd64.tar.gz"
    sha256 "288aaafdcc79be8063eb56163be3d82d33cf1ec720fd6a08e6f9bb3f0d28ba80"
  elsif OS.linux? && Hardware::CPU.arm?
    url "https://github.com/unix2dos/ww/releases/download/v0.6.0/ww-v0.6.0-linux-arm64.tar.gz"
    sha256 "cf415a7c4631e4d56fe18596e2e814209d0f704fa54390d910f7f140fdd4198a"
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/unix2dos/ww/releases/download/v0.6.0/ww-v0.6.0-linux-amd64.tar.gz"
    sha256 "afab5955258e4e8b3bb1927dec5a5cc2a42a5395c3b1bef5d86f1ed9cd1c42e4"
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

      Add these lines to your shell rc file:

        export WW_HELPER_BIN="#{opt_bin}/ww-helper"
        source "#{opt_libexec}/ww.sh"

      For zsh:
        echo 'export WW_HELPER_BIN="#{opt_bin}/ww-helper"' >> ~/.zshrc
        echo 'source "#{opt_libexec}/ww.sh"' >> ~/.zshrc

      For bash:
        echo 'export WW_HELPER_BIN="#{opt_bin}/ww-helper"' >> ~/.bashrc
        echo 'source "#{opt_libexec}/ww.sh"' >> ~/.bashrc
    EOS
  end

  test do
    assert_path_exists libexec/"ww.sh"
    assert_match "Usage: ww-helper", shell_output("#{bin}/ww-helper help")
  end
end
