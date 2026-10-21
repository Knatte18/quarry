namespace Loomyard.Engine;

public interface IProvider<T>
{
    int Value { get; }
}

public class A { }

public class B { }

public partial class Renderer : IProvider<A>, IProvider<B>, System.IDisposable
{
    public Renderer() { }

    static Renderer() { }

    ~Renderer() { }

    public void Draw(int count) { }

    public void Draw(double scale) { }

    public void Draw() { }

    public int this[int index] => index;

    public static Renderer operator +(Renderer left, Renderer right) => left;

    public static bool operator <(Renderer left, Renderer right) => false;

    public static bool operator >(Renderer left, Renderer right) => false;

    public static int operator >>(Renderer left, int shift) => shift;

    public static implicit operator System.Text.Rune(Renderer value) => default;

    public string Name { get; set; }

    public int Width { get; }

    public int X, Y;

    public const int Limit = 10;

    public event System.EventHandler Changed;

    public event System.EventHandler Resized { add { } remove { } }

    int IProvider<A>.Value => 1;

    int IProvider<B>.Value => 2;

    void System.IDisposable.Dispose() { }

    public class Cache
    {
        public T Get<T>(string key) => default;
    }

    partial void OnDraw();
}
