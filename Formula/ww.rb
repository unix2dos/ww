class Ww < Formula
  desc "Worktree primitive your AI agents and you share"
  homepage "https://github.com/unix2dos/ww"
  version "0.11.1"

  if OS.mac? && Hardware::CPU.arm?
    url "https://github.com/unix2dos/ww/releases/download/v0.11.1/ww-v0.11.1-darwin-arm64.tar.gz"
    sha256 "e04a5f8e8a325d2b65142501957f31a4f0f9fcb9b7c862dca00ae6bce513e9cb"
  elsif OS.mac? && Hardware::CPU.intel?
    url "https://github.com/unix2dos/ww/releases/download/v0.11.1/ww-v0.11.1-darwin-amd64.tar.gz"
    sha256 "f6a91870b4721538b51fd18c1c7f21b57f3fff584d8857aa97d8c2b092034d02"
  elsif OS.linux? && Hardware::CPU.arm?
    url "https://github.com/unix2dos/ww/releases/download/v0.11.1/ww-v0.11.1-linux-arm64.tar.gz"
    sha256 "23c06290126f8e56e61f4c0b73efebc0a3ee995c2e3c5a36e3c85f704c628059"
  elsif OS.linux? && Hardware::CPU.intel?
    url "https://github.com/unix2dos/ww/releases/download/v0.11.1/ww-v0.11.1-linux-amd64.tar.gz"
    sha256 "2cffb5d9e5b0063c6514ba81d82390217d773a1fe243eb8ef1e02b5ae87e7517"
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
